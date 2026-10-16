package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"
)

// gameYAMLWith renders a game with raw extra flow-mapping fields.
func gameYAMLWith(name, saves, fields string) string {
	return fmt.Sprintf("  - {name: %s, saves: ['%s'], settle_seconds: 15, %s}\n", name, saves, fields)
}

func TestRunningByInstallDir(t *testing.T) {
	gameDir := filepath.Join(t.TempDir(), "Game")
	h := newHarness(t, "", func(h *harness) {
		writeFile(t, h.cfg, configYAML(h.root,
			gameYAMLWith("Test Game", h.saves, fmt.Sprintf("install_dir: '%s', processes: [nope.exe]", gameDir))))
	})
	h.tick(0)
	h.fake.SetRunningPaths(filepath.Join(gameDir, "bin", "any.exe"))
	h.tick(3)
	if got := h.kinds(t); len(got) != 1 {
		t.Fatalf("archives = %v, want pre", got)
	}
	if st := h.app.Status(); !slices.Equal(st.Running, []string{"Test Game"}) {
		t.Errorf("status = %+v", st)
	}
	h.fake.SetRunningPaths(filepath.Join(filepath.Dir(gameDir), "Other", "any.exe"))
	h.tick(3)
	if st := h.app.Status(); len(st.Running) != 0 {
		t.Errorf("exe outside install_dir counted as running: %+v", st)
	}
}
