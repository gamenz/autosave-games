// Command autosave-tray watches games from the system tray (build with
// -ldflags "-H windowsgui" so no console window appears).
package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"autosave-games/internal/app"
	"autosave-games/internal/config"
	"autosave-games/internal/logging"
	"autosave-games/internal/notify"
	"autosave-games/internal/process"
	"autosave-games/internal/tray"
)

var version = "dev"

// startResult is what start leaves for main once the tray has exited.
type startResult struct {
	log    *slog.Logger
	closer io.Closer
	failed bool
}

func main() {
	release, already, err := tray.AcquireSingleInstance(tray.MutexName)
	if err == nil && already {
		return
	}
	if release != nil {
		defer release()
	}

	logDir := logging.DefaultDir()
	cfgPath := config.DefaultPath()
	done := make(chan startResult, 1)
	// The tray message loop starts before any slow setup: Windows (and the
	// Start menu, which stays open until then) sees the app as launched as
	// soon as that loop runs.
	tray.Run(tray.Options{
		ConfigPath: cfgPath,
		LogPath:    filepath.Join(logDir, logging.FileName),
		Start: func() (tray.Started, bool) {
			s, res := start(logDir, cfgPath)
			done <- res
			return s, !res.failed
		},
	})

	var res startResult
	select {
	case res = <-done:
	default:
	}
	if res.failed {
		if res.closer != nil {
			res.closer.Close()
		}
		os.Exit(1)
	}
	if res.log != nil {
		res.log.Info("tray stopped")
	}
	if res.closer != nil {
		res.closer.Close()
	}
}

// start sets up logging and the watcher loop; it runs from the tray's
// ready callback.
func start(logDir, cfgPath string) (tray.Started, startResult) {
	var res startResult
	log, closer, err := logging.Setup(logDir)
	if err != nil {
		log = slog.New(slog.DiscardHandler)
	} else {
		res.closer = closer
	}
	res.log = log
	log.Info("tray starting", "version", version)

	n := notify.NewNotifier(config.DefaultNotify, notify.ToastSender{}, log)
	a, err := app.New(app.Deps{
		ConfigPath: cfgPath, Lister: process.NewSystemLister(), Notifier: n,
		Log: log, Now: time.Now, AppVersion: version,
	})
	if err != nil {
		n.Critical("autosave-games cannot start", "Fix "+cfgPath+": "+err.Error())
		res.failed = true
		return tray.Started{}, res
	}

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		a.Run(ctx)
		close(stopped)
	}()
	onQuit := func() {
		cancel()
		<-stopped
	}
	return tray.Started{App: a, Notifier: n, OnQuit: onQuit}, res
}
