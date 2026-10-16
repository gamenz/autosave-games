package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"autosave-games/configs"
)

func TestEnsureConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "autosave-games", "games.yaml")
	created, err := EnsureConfig(p)
	if err != nil || !created {
		t.Fatalf("EnsureConfig = %v, %v", created, err)
	}
	got, _ := os.ReadFile(p)
	if !bytes.Equal(got, configs.Example) {
		t.Error("content differs from embedded example")
	}
	writeFile(t, p, "custom")
	created, err = EnsureConfig(p)
	if err != nil || created {
		t.Fatalf("second EnsureConfig = %v, %v", created, err)
	}
	if got, _ := os.ReadFile(p); string(got) != "custom" {
		t.Error("existing config overwritten")
	}
}
