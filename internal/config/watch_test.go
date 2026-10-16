package config

import (
	"os"
	"path/filepath"
	"testing"

	"autosave-games/configs"
)

func TestFileWatcher(t *testing.T) {
	p := filepath.Join(t.TempDir(), "games.yaml")
	if err := os.WriteFile(p, []byte("a: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := NewFileWatcher(p)
	if w.Changed() {
		t.Error("first call must record state and return false")
	}
	if err := os.WriteFile(p, []byte("a: 1\nb: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !w.Changed() {
		t.Error("rewrite not detected")
	}
	if w.Changed() {
		t.Error("no change since last call")
	}
}

func TestExampleConfigIsValid(t *testing.T) {
	c, err := Parse(configs.Example, testEnv)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(c); err != nil {
		t.Fatalf("example config invalid: %v", err)
	}
}
