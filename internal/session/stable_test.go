package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func always(string) bool { return true }

func TestStableUnchanged(t *testing.T) {
	dir := t.TempDir()
	a, b := writeFile(t, dir, "a.sav", "1"), writeFile(t, dir, "b.sav", "2")
	prev, err := StatFiles([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	cur, _ := StatFiles([]string{a, b})
	if !Stable(prev, cur, always) {
		t.Error("unchanged files should be stable")
	}
	if Stable(nil, cur, always) {
		t.Error("first observation must not be stable")
	}
}

func TestStableDetectsChange(t *testing.T) {
	dir := t.TempDir()
	a, b := writeFile(t, dir, "a.sav", "1"), writeFile(t, dir, "b.sav", "2")
	prev, _ := StatFiles([]string{a, b})

	writeFile(t, dir, "a.sav", "12")
	os.Chtimes(a, time.Now(), time.Now().Add(time.Second))
	cur, _ := StatFiles([]string{a, b})
	if Stable(prev, cur, always) {
		t.Error("modified file should be unstable")
	}

	prev, _ = StatFiles([]string{a, b})
	cur, _ = StatFiles([]string{a})
	if Stable(prev, cur, always) {
		t.Error("changed file set should be unstable")
	}

	cur, _ = StatFiles([]string{a, b})
	if Stable(prev, cur, func(p string) bool { return p != b }) {
		t.Error("unopenable file should be unstable")
	}
}

func TestStatFilesMissing(t *testing.T) {
	if _, err := StatFiles([]string{filepath.Join(t.TempDir(), "nope")}); err == nil {
		t.Error("expected error for missing file")
	}
}
