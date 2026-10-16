package backup

import (
	"os"
	"path/filepath"
	"testing"

	"autosave-games/internal/config"
)

func hashDir(t *testing.T, dir string) string {
	t.Helper()
	entries, errs := Collect([]config.SaveBase{{AbsBase: dir}}, nil)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	h, err := ContentHash(entries)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestContentHash(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.sav"), "hello")
	writeFile(t, filepath.Join(dir, "b.sav"), "world")
	h1, h2 := hashDir(t, dir), hashDir(t, dir)
	if h1 != h2 || len(h1) != 64 {
		t.Fatalf("hash not stable: %q vs %q", h1, h2)
	}

	writeFile(t, filepath.Join(dir, "a.sav"), "hellO")
	h3 := hashDir(t, dir)
	if h3 == h1 {
		t.Error("content change not detected")
	}

	if err := os.Rename(filepath.Join(dir, "a.sav"), filepath.Join(dir, "c.sav")); err != nil {
		t.Fatal(err)
	}
	if hashDir(t, dir) == h3 {
		t.Error("rename not detected")
	}
}
