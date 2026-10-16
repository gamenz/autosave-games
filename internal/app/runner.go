package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"autosave-games/internal/backup"
	"autosave-games/internal/config"
	"autosave-games/internal/process"
	"autosave-games/internal/session"
)

// runner is the per-game session state. Only Tick (under App.mu) touches it.
type runner struct {
	game      config.Game
	machine   *session.Machine
	prevStat  map[string]session.FileStat
	lastCheck time.Time
	busy      int  // in-flight backup workers
	removed   bool // dropped from config; kept until its session ends
}

type doneMsg struct {
	slug   string
	name   string
	kind   backup.Kind
	result backup.Result
	err    error
}

func (a *App) stepRunner(r *runner, snap process.Snapshot, now time.Time) {
	running := snap.Running(r.game.Processes, r.game.InstallDir)
	stable := false
	if r.machine.State() == session.WaitingStable && now.Sub(r.lastCheck) >= session.StableInterval {
		cur, err := session.StatFiles(a.service.Files(r.game))
		if err == nil {
			stable = session.Stable(r.prevStat, cur, session.CanOpen)
		}
		r.prevStat, r.lastCheck = cur, now
	}
	before := r.machine.State()
	action := r.machine.Step(session.Input{Now: now, Running: running, Stable: stable})
	if after := r.machine.State(); after != before {
		attrs := []any{"game", r.game.Name, "from", before.String(), "to", after.String()}
		if after == session.BackingUp {
			attrs = append(attrs, "stable", stable)
		}
		a.deps.Log.Info("session state", attrs...)
	}
	switch action {
	case session.SnapshotPre:
		a.startBackup(r, backup.KindPre)
	case session.BackupPost:
		a.startBackup(r, backup.KindPost)
	}
	if r.machine.State() != session.WaitingStable {
		r.prevStat, r.lastCheck = nil, time.Time{}
	}
}

// startBackup runs one backup on a worker goroutine with a snapshot of the
// service, so config reloads never race with running workers.
func (a *App) startBackup(r *runner, kind backup.Kind) {
	r.busy++
	svc := *a.service
	g := r.game
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		res, err := svc.Run(g, kind)
		a.done <- doneMsg{slug: g.Slug, name: g.Name, kind: kind, result: res, err: err}
	}()
}

func (a *App) drainDone() {
	for {
		select {
		case m := <-a.done:
			a.handleDone(m)
		default:
			return
		}
	}
}

func (a *App) handleDone(m doneMsg) {
	if r := a.runners[m.slug]; r != nil {
		r.busy--
		if m.kind == backup.KindPost {
			r.machine.BackupDone()
			a.deps.Log.Info("session state", "game", m.name,
				"from", session.BackingUp.String(), "to", r.machine.State().String())
		}
		a.dropIfFinished(m.slug, r)
	}
	a.report(m)
}

func (a *App) report(m doneMsg) {
	n, log := a.deps.Notifier, a.deps.Log
	switch {
	case errors.Is(m.err, backup.ErrLocked):
		n.Error("Backup skipped: "+m.name, m.err.Error())
	case m.err != nil:
		n.Critical("Backup failed: "+m.name, m.err.Error())
	case m.result.Skipped && m.result.Reason == backup.ReasonNoFiles:
		n.Error("No save files: "+m.name, fmt.Sprint(errors.Join(m.result.Warnings...)))
	case m.result.Skipped:
		log.Info("backup skipped", "game", m.name, "kind", m.kind, "reason", m.result.Reason)
	default:
		n.Info("Backed up "+m.name, string(m.kind)+" · "+filepath.Base(m.result.Path))
		for _, w := range m.result.Warnings {
			log.Warn("backup warning", "game", m.name, "err", w)
		}
	}
}

func (a *App) dropIfFinished(slug string, r *runner) {
	if r.removed && r.busy == 0 && r.machine.State() == session.Idle {
		delete(a.runners, slug)
	}
}
