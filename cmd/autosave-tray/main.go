// Command autosave-tray watches games from the system tray (build with
// -ldflags "-H windowsgui" so no console window appears).
package main

import (
	"context"
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

func main() {
	release, already, err := tray.AcquireSingleInstance(tray.MutexName)
	if err == nil && already {
		return
	}
	if release != nil {
		defer release()
	}

	logDir := logging.DefaultDir()
	log, closer, err := logging.Setup(logDir)
	if err != nil {
		log = slog.New(slog.DiscardHandler)
	} else {
		defer closer.Close()
	}
	log.Info("tray starting", "version", version)

	cfgPath := config.DefaultPath()
	n := notify.NewNotifier(config.DefaultNotify, notify.ToastSender{}, log)
	a, err := app.New(app.Deps{
		ConfigPath: cfgPath, Lister: process.NewSystemLister(), Notifier: n,
		Log: log, Now: time.Now, AppVersion: version,
	})
	if err != nil {
		n.Critical("autosave-games cannot start", "Fix "+cfgPath+": "+err.Error())
		if closer != nil {
			closer.Close()
		}
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		a.Run(ctx)
		close(stopped)
	}()
	tray.Run(a, tray.Options{
		ConfigPath: cfgPath,
		LogPath:    filepath.Join(logDir, logging.FileName),
		OnQuit: func() {
			cancel()
			<-stopped
		},
	})
	log.Info("tray stopped")
}
