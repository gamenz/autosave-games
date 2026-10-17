package tray

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/systray"

	"autosave-games/internal/app"
	"autosave-games/internal/notify"
)

// Options are the paths and the startup hook the menu needs.
type Options struct {
	ConfigPath string
	LogPath    string
	// Start runs once the tray message loop is up, so slow setup does not
	// delay the moment Windows sees the app as launched. ok=false quits.
	Start func() (s Started, ok bool)
}

// Started is what Start hands to the menu.
type Started struct {
	App      *app.App
	Notifier *notify.Notifier // scan results (always shown)
	OnQuit   func()           // stops the app loop; called before the tray exits
}

const refreshInterval = 2 * time.Second

type menu struct {
	app       *app.App
	opts      Options
	started   Started
	status    *systray.MenuItem
	backupNow *systray.MenuItem
	gameItems []*systray.MenuItem
	gameKey   string
	stopGames chan struct{}
	refreshMu sync.Mutex // refresh runs from the menu loop and scan goroutines
}

// Run shows the tray icon, then calls opts.Start; it blocks until Quit is
// chosen or Start fails.
func Run(opts Options) {
	m := &menu{opts: opts}
	systray.Run(m.onReady, nil)
}

func (m *menu) onReady() {
	systray.SetIcon(iconData)
	systray.SetTooltip("Autosave Games")
	m.status = systray.AddMenuItem("Starting…", "")
	m.status.Disable()
	systray.AddSeparator()
	m.backupNow = systray.AddMenuItem("Backup now", "Back up a game immediately")
	openBackups := systray.AddMenuItem("Open backup folder", "")
	editConfig := systray.AddMenuItem("Edit config", "")
	scanSteam := systray.AddMenuItem("Scan Steam games", "Add installed Steam games to the config")
	reload := systray.AddMenuItem("Reload config", "")
	openLog := systray.AddMenuItem("Open log", "")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Stop watching games")

	started, ok := m.opts.Start()
	if !ok {
		systray.Quit()
		return
	}
	m.started, m.app = started, started.App
	m.refresh()

	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.refresh()
			case <-openBackups.ClickedCh:
				root := m.app.Config().BackupRoot
				os.MkdirAll(root, 0o755)
				exec.Command("explorer.exe", root).Start()
			case <-editConfig.ClickedCh:
				exec.Command("notepad.exe", m.opts.ConfigPath).Start()
			case <-scanSteam.ClickedCh:
				scanSteam.Disable()
				go m.scanSteam(scanSteam)
			case <-reload.ClickedCh:
				m.app.ReloadConfig()
				m.refresh()
			case <-openLog.ClickedCh:
				exec.Command("notepad.exe", m.opts.LogPath).Start()
			case <-quit.ClickedCh:
				if m.started.OnQuit != nil {
					m.started.OnQuit()
				}
				systray.Quit()
				return
			}
		}
	}()
}

// refresh updates the status line and rebuilds the "Backup now" submenu
// when the configured games change.
func (m *menu) refresh() {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()
	st := m.app.Status()
	running := "Idle"
	if len(st.Running) > 0 {
		running = "Running: " + strings.Join(st.Running, ", ")
	}
	m.status.SetTitle("Watching " + strconv.Itoa(st.Watching) + " games · " + running)

	games := m.app.Games()
	keys := make([]string, len(games))
	for i, g := range games {
		keys[i] = g.Slug + "=" + g.Name
	}
	key := strings.Join(keys, "|")
	if key == m.gameKey {
		return
	}
	m.gameKey = key
	if m.stopGames != nil {
		close(m.stopGames)
	}
	for _, it := range m.gameItems {
		it.Remove()
	}
	m.gameItems = nil
	m.stopGames = make(chan struct{})
	for _, g := range games {
		it := m.backupNow.AddSubMenuItem(g.Name, "")
		m.gameItems = append(m.gameItems, it)
		go waitClick(it, m.stopGames, func(slug string) func() {
			return func() { m.app.BackupNow(slug) }
		}(g.Slug))
	}
}

func waitClick(it *systray.MenuItem, stop <-chan struct{}, onClick func()) {
	for {
		select {
		case <-it.ClickedCh:
			onClick()
		case <-stop:
			return
		}
	}
}

// scanSteam runs a Steam scan off the menu loop and reports the result.
func (m *menu) scanSteam(item *systray.MenuItem) {
	defer item.Enable()
	sum, err := m.app.ScanSteam()
	if n := m.started.Notifier; n != nil {
		if err != nil {
			n.Notice("Steam scan failed", err.Error())
		} else {
			n.Notice("Steam scan", app.FormatScanSummary(sum))
		}
	}
	m.refresh()
}
