package appdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirIsExecutableFolder(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exe, _ = filepath.EvalSymlinks(exe)
	if got, want := Dir(), filepath.Dir(exe); got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}
