package config

import (
	"path/filepath"
	"testing"

	"autosave-games/internal/appdir"
)

func TestDefaultPathNextToExecutable(t *testing.T) {
	if got, want := DefaultPath(), filepath.Join(appdir.Dir(), "games.yaml"); got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}
