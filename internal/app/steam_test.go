package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"autosave-games/internal/config"
	"autosave-games/internal/notify"
	"autosave-games/internal/steam"
)

func writeSteamTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "steamapps", "appmanifest_10.acf"),
		"\"AppState\" { \"appid\" \"10\" \"name\" \"Game X\" \"installdir\" \"GameX\" }")
	writeFile(t, filepath.Join(root, "steamapps", "common", "GameX", "x.exe"), "")
	writeFile(t, filepath.Join(root, "userdata", "111", "10", "remotecache.vdf"),
		"\"10\" { \"GX/Saved/a.sav\" { \"root\" \"3\" } }")
	return root
}

func newSteamApp(t *testing.T, root func() (string, error)) *harness {
	t.Helper()
	h := newHarness(t, "", nil)
	log := h.app.deps.Log
	a, err := New(Deps{ConfigPath: h.cfg, Lister: h.fake, Notifier: notify.NewNotifier("all", h.rec, log),
		Log: log, Now: func() time.Time { return h.now }, AppVersion: "test", SteamRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	h.app = a
	return h
}

func TestScanSteamAddsGame(t *testing.T) {
	root := writeSteamTree(t)
	h := newSteamApp(t, func() (string, error) { return root, nil })
	sum, err := h.app.ScanSteam()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sum.Added, []string{"Game X"}) {
		t.Fatalf("summary = %+v", sum)
	}
	if w := h.app.Status().Watching; w != 2 {
		t.Errorf("Watching = %d, want 2", w)
	}
	c, err := config.Load(h.cfg)
	if err != nil || len(c.Games) != 2 || c.Games[1].SteamAppID != 10 {
		t.Fatalf("config = %+v, %v", c, err)
	}
}

func TestScanSteamNotFound(t *testing.T) {
	h := newSteamApp(t, func() (string, error) { return "", steam.ErrNotFound })
	before := readConfigText(t, h.cfg)
	if _, err := h.app.ScanSteam(); !errors.Is(err, steam.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if readConfigText(t, h.cfg) != before {
		t.Error("config changed")
	}
}

func TestFormatScanSummary(t *testing.T) {
	s := config.AddSummary{Added: []string{"A", "B"}, NeedSaves: []string{"C"}}
	if got, want := FormatScanSummary(s), "added 2 games, 1 need save paths, 0 already configured"; got != want {
		t.Errorf("FormatScanSummary = %q, want %q", got, want)
	}
}

func readConfigText(t *testing.T, p string) string {
	t.Helper()
	c, err := config.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprint(c.Games)
}
