// Package app runs the watcher loop: process polling, session state
// machines, backups, config hot reload and notifications.
package app

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"autosave-games/internal/backup"
	"autosave-games/internal/config"
	"autosave-games/internal/notify"
	"autosave-games/internal/process"
	"autosave-games/internal/session"
)

// Deps are the App's collaborators.
type Deps struct {
	ConfigPath string
	Lister     process.Lister
	Notifier   *notify.Notifier
	Log        *slog.Logger
	Now        func() time.Time
	AppVersion string
	SteamRoot  func() (string, error) // nil: steam.LocateRoot
}

// Status is a snapshot for the tray menu.
type Status struct {
	Watching int
	Running  []string
}

// App owns the runners. All mutable state is guarded by mu.
type App struct {
	deps    Deps
	watcher *config.FileWatcher
	done    chan doneMsg
	wg      sync.WaitGroup

	mu      sync.Mutex
	cfg     *config.Config
	service *backup.Service
	runners map[string]*runner
}

// New creates the config on first run, loads it and prepares runners.
func New(d Deps) (*App, error) {
	created, err := EnsureConfig(d.ConfigPath)
	if err != nil {
		return nil, err
	}
	if created {
		d.Notifier.Error("Config created", "Edit "+d.ConfigPath+" to add your games")
	}
	cfg, err := config.Load(d.ConfigPath)
	if err != nil {
		return nil, err
	}
	a := &App{
		deps:    d,
		watcher: config.NewFileWatcher(d.ConfigPath),
		done:    make(chan doneMsg, 64),
		service: &backup.Service{AppVersion: d.AppVersion, Now: d.Now},
		runners: map[string]*runner{},
	}
	a.watcher.Changed()
	snap, err := d.Lister.Snapshot()
	if err != nil {
		d.Log.Warn("list processes", "err", err)
	}
	a.applyConfig(cfg, snap)
	return a, nil
}

// Tick runs one loop iteration.
func (a *App) Tick() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.drainDone()
	snap, listErr := a.deps.Lister.Snapshot()
	if a.watcher.Changed() {
		a.reloadLocked(snap)
	}
	if listErr != nil {
		// A failed listing must not look like "game exited".
		a.deps.Log.Warn("list processes", "err", listErr)
		return
	}
	now := a.deps.Now()
	for slug, r := range a.runners {
		a.stepRunner(r, snap, now)
		a.dropIfFinished(slug, r)
	}
}

// Run ticks until ctx is cancelled, then waits for in-flight backups.
func (a *App) Run(ctx context.Context) {
	for {
		a.Tick()
		select {
		case <-ctx.Done():
			a.Wait()
			a.mu.Lock()
			a.drainDone()
			a.mu.Unlock()
			return
		case <-time.After(a.NextDelay()):
		}
	}
}

// Wait blocks until all started backup workers have finished.
func (a *App) Wait() { a.wg.Wait() }

// NextDelay is the stability interval while any game waits for its files
// to settle, otherwise poll_seconds.
func (a *App) NextDelay() time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, r := range a.runners {
		if r.machine.State() == session.WaitingStable {
			return session.StableInterval
		}
	}
	return time.Duration(a.cfg.PollSeconds) * time.Second
}

// Status reports how many games are watched and which are running.
func (a *App) Status() Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	var st Status
	for _, r := range a.runners {
		if r.removed {
			continue
		}
		st.Watching++
		if r.machine.State() == session.Running {
			st.Running = append(st.Running, r.game.Name)
		}
	}
	sort.Strings(st.Running)
	return st
}

// Games returns the configured games.
func (a *App) Games() []config.Game {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]config.Game(nil), a.cfg.Games...)
}

// Config returns the active configuration.
func (a *App) Config() *config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

// BackupNow starts a manual backup of the game with the given slug.
func (a *App) BackupNow(slug string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if r := a.runners[slug]; r != nil && !r.removed {
		a.startBackup(r, backup.KindManual)
	}
}

// ReloadConfig re-reads the config file now.
func (a *App) ReloadConfig() {
	snap, _ := a.deps.Lister.Snapshot()
	a.mu.Lock()
	defer a.mu.Unlock()
	a.reloadLocked(snap)
}

func (a *App) reloadLocked(snap process.Snapshot) {
	cfg, err := config.Load(a.deps.ConfigPath)
	if err != nil {
		a.deps.Notifier.Error("Config error", err.Error())
		return
	}
	a.applyConfig(cfg, snap)
	a.deps.Log.Info("config reloaded", "games", len(cfg.Games))
}

// applyConfig adds runners for new games, updates existing ones in place
// and marks missing ones removed so their session can finish.
func (a *App) applyConfig(cfg *config.Config, snap process.Snapshot) {
	seen := map[string]bool{}
	for _, g := range cfg.Games {
		seen[g.Slug] = true
		if r := a.runners[g.Slug]; r != nil {
			r.game, r.removed = g, false
			continue
		}
		settle := time.Duration(g.SettleSeconds) * time.Second
		a.runners[g.Slug] = &runner{game: g, machine: session.NewMachine(settle, snap.Running(g.Processes, g.InstallDir))}
	}
	for slug, r := range a.runners {
		if !seen[slug] {
			r.removed = true
			a.dropIfFinished(slug, r)
		}
	}
	a.cfg = cfg
	a.service.Root = cfg.BackupRoot
	a.deps.Notifier.SetLevel(cfg.Notify)
}
