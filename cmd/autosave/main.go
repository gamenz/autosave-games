// Command autosave is the CLI: list, back up and restore game saves.
package main

import (
	"log/slog"
	"os"
	"time"

	"autosave-games/internal/cli"
	"autosave-games/internal/logging"
	"autosave-games/internal/process"
)

var version = "dev"

func main() {
	log, closer, err := logging.Setup(logging.DefaultDir())
	if err != nil {
		log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}
	code := cli.Run(os.Args[1:], cli.Env{
		Stdout: os.Stdout, Stderr: os.Stderr, Stdin: os.Stdin,
		Lister: process.NewSystemLister(), Log: log, Now: time.Now, Version: version,
	})
	if closer != nil {
		closer.Close()
	}
	os.Exit(code)
}
