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

// Options are the paths and callbacks the menu needs.
type Options struct {
	ConfigPath string
	LogPath    string
	Notifier   *notify.Notifier // scan results (always shown)
	OnQuit     func()           // stops the app loop; called before the tray exits
}

const refreshInterval = 2 * time.Second

type menu struct {
	app       *app.App
	opts      Options
	status    *systray.MenuItem
	backupNow *systray.MenuItem
	gameItems []*systray.MenuItem
	gameKey   string
	stopGames chan struct{}
	refreshMu sync.Mutex // refresh runs from the menu loop and scan goroutines
}

// Run shows the tray icon and blocks until Quit is chosen.
func Run(a *app.App, opts Options) {
	m := &menu{app: a, opts: opts}
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
				if m.opts.OnQuit != nil {
					m.opts.OnQuit()
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
	if m.opts.Notifier != nil {
		if err != nil {
			m.opts.Notifier.Notice("Steam scan failed", err.Error())
		} else {
			m.opts.Notifier.Notice("Steam scan", app.FormatScanSummary(sum))
		}
	}
	m.refresh()
}
