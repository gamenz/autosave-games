package process

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnyRunning(t *testing.T) {
	names := map[string]bool{"eldenring.exe": true}
	if !AnyRunning(names, []string{"EldenRing.exe"}) {
		t.Error("case-insensitive match failed")
	}
	if !AnyRunning(names, []string{"start_protected_game.exe", "eldenring.exe"}) {
		t.Error("second process should match")
	}
	if AnyRunning(names, []string{"hk.exe"}) {
		t.Error("hk.exe should not match")
	}
}

func TestSystemListerSeesSelf(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	names, err := NewSystemLister().Names()
	if err != nil {
		t.Fatalf("Names: %v", err)
	}
	if self := strings.ToLower(filepath.Base(exe)); !names[self] {
		t.Errorf("own process %q not listed (%d names)", self, len(names))
	}
}

func TestFake(t *testing.T) {
	f := NewFake()
	f.SetRunning("Game.exe")
	names, err := f.Names()
	if err != nil || !names["game.exe"] {
		t.Fatalf("Names = %v, %v", names, err)
	}
	want := errors.New("boom")
	f.SetErr(want)
	if _, err := f.Names(); !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
}
