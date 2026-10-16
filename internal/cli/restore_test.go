package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"autosave-games/internal/backup"
)

func TestRestoreRefusesWhileRunning(t *testing.T) {
	f := newFixture(t)
	f.run("backup", "test-game")
	writeFile(t, filepath.Join(f.saves, "a.sav"), "v2")
	f.fake.SetRunning("game.exe")
	code := f.run("restore", "test-game", "--yes")
	if code != 1 || !strings.Contains(f.stderr.String(), `"Test Game" is running; close it first`) {
		t.Fatalf("exit %d, stderr %q", code, f.stderr)
	}
	if got := readFile(t, filepath.Join(f.saves, "a.sav")); got != "v2" {
		t.Errorf("a.sav = %q, must be unchanged", got)
	}
}

func TestRestoreAbort(t *testing.T) {
	f := newFixture(t)
	f.run("backup", "test-game")
	writeFile(t, filepath.Join(f.saves, "a.sav"), "v2")
	f.stdin = "n\n"
	code := f.run("restore", "test-game")
	if code != 1 || !strings.Contains(f.stdout.String(), "aborted") {
		t.Fatalf("exit %d, stdout %q", code, f.stdout)
	}
	if got := readFile(t, filepath.Join(f.saves, "a.sav")); got != "v2" {
		t.Errorf("a.sav = %q", got)
	}
}

func TestRestoreConfirmed(t *testing.T) {
	f := newFixture(t)
	f.run("backup", "test-game")
	writeFile(t, filepath.Join(f.saves, "a.sav"), "v2")
	f.stdin = "y\n"
	if code := f.run("restore", "test-game"); code != 0 {
		t.Fatalf("exit %d: %s", code, f.stderr)
	}
	if got := readFile(t, filepath.Join(f.saves, "a.sav")); got != "v1" {
		t.Errorf("a.sav = %q, want v1", got)
	}
	if !strings.Contains(f.stdout.String(), "Restored 1 files") {
		t.Errorf("stdout %q", f.stdout)
	}
	found := false
	for _, a := range f.archives(t) {
		found = found || a.Kind == backup.KindPreRestore
	}
	if !found {
		t.Error("no prerestore archive")
	}
}

func TestRestoreAt(t *testing.T) {
	f := newFixture(t)
	f.run("backup", "test-game")
	older := f.archives(t)[0].Name
	writeFile(t, filepath.Join(f.saves, "a.sav"), "v2")
	f.run("backup", "test-game")
	writeFile(t, filepath.Join(f.saves, "a.sav"), "v3")
	if code := f.run("restore", "test-game", "--at", strings.TrimSuffix(older, ".zip"), "--yes"); code != 0 {
		t.Fatalf("exit %d: %s", code, f.stderr)
	}
	if got := readFile(t, filepath.Join(f.saves, "a.sav")); got != "v1" {
		t.Errorf("a.sav = %q, want v1", got)
	}
}
