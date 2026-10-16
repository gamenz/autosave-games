// Package logging configures the shared rotating log file.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// FileName is the active log file name inside the log directory.
const FileName = "autosave.log"

// DefaultDir is %LOCALAPPDATA%\autosave-games\logs.
func DefaultDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "autosave-games", "logs")
}

// Setup returns a text logger writing to dir\autosave.log, rotated at 5 MB
// with 3 old files kept. Close the returned Closer on exit.
func Setup(dir string) (*slog.Logger, io.Closer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create log dir: %w", err)
	}
	w := &lumberjack.Logger{Filename: filepath.Join(dir, FileName), MaxSize: 5, MaxBackups: 3}
	return slog.New(slog.NewTextHandler(w, nil)), w, nil
}
