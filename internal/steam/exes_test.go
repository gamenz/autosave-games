package steam

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"
)

func TestExecutables(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{
		"modengine2_launcher.exe", "Game/eldenring.exe", "Game/start_protected_game.exe",
		"Game/EasyAntiCheat/easyanticheat_eos_setup.exe", "modengine2/crashpad/crashpad_handler.exe",
		"UnityCrashHandler64.exe", "unins000.exe", "Redist/vc_redist.x64.exe", "readme.txt",
		"a/b/c/d/too_deep.exe", "Game/ELDENRING.EXE",
	} {
		writeFile(t, filepath.Join(dir, filepath.FromSlash(f)), "")
	}
	got, err := Executables(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"modengine2_launcher.exe", "eldenring.exe", "start_protected_game.exe"}; !slices.Equal(got, want) {
		t.Fatalf("Executables = %v, want %v", got, want)
	}
}

func TestExecutablesCap(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 12; i++ {
		writeFile(t, filepath.Join(dir, fmt.Sprintf("tool%02d.exe", i)), "")
	}
	if got, _ := Executables(dir); len(got) != 10 || got[0] != "tool00.exe" {
		t.Fatalf("Executables = %v", got)
	}
}

func TestExecutablesMissing(t *testing.T) {
	if _, err := Executables(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error")
	}
}
