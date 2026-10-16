package app

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"autosave-games/internal/backup"
	"autosave-games/internal/notify"
	"autosave-games/internal/process"
	"autosave-games/internal/session"
)

type harness struct {
	app              *App
	fake             *process.Fake
	rec              *notify.RecordingSender
	logs             *syncBuffer
	now              time.Time
	saves, root, cfg string
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func configYAML(root string, games ...string) string {
	return fmt.Sprintf("backup_root: '%s'\nnotify: all\ngames:\n%s", root, strings.Join(games, ""))
}

func gameYAML(name, exe, saves string) string {
	return fmt.Sprintf("  - {name: %s, processes: [%s], saves: ['%s'], settle_seconds: 15}\n", name, exe, saves)
}

// newHarness builds an App over temp dirs; root "" means a fresh temp dir.
func newHarness(t *testing.T, root string, setup func(*harness)) *harness {
	t.Helper()
	h := &harness{fake: process.NewFake(), rec: &notify.RecordingSender{}, logs: &syncBuffer{},
		now: time.Date(2026, 10, 17, 20, 0, 0, 0, time.Local), saves: t.TempDir(), root: root}
	if h.root == "" {
		h.root = t.TempDir()
	}
	h.cfg = filepath.Join(t.TempDir(), "games.yaml")
	writeFile(t, filepath.Join(h.saves, "a.sav"), "v1")
	writeFile(t, h.cfg, configYAML(h.root, gameYAML("Test Game", "game.exe", h.saves)))
	if setup != nil {
		setup(h)
	}
	log := slog.New(slog.NewTextHandler(h.logs, nil))
	a, err := New(Deps{ConfigPath: h.cfg, Lister: h.fake, Notifier: notify.NewNotifier("all", h.rec, log),
		Log: log, Now: func() time.Time { return h.now }, AppVersion: "test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h.app = a
	return h
}

func (h *harness) tick(sec int) {
	h.now = h.now.Add(time.Duration(sec) * time.Second)
	h.app.Tick()
	h.app.Wait()
}

func (h *harness) kinds(t *testing.T) []backup.Kind {
	t.Helper()
	list, err := backup.List(filepath.Join(h.root, "test-game"))
	if err != nil {
		t.Fatal(err)
	}
	var out []backup.Kind
	for _, a := range list {
		out = append(out, a.Kind)
	}
	return out
}

func (h *harness) countMessages(prefix string) int {
	n := 0
	for _, m := range h.rec.Messages() {
		if strings.HasPrefix(m, prefix) {
			n++
		}
	}
	return n
}

func (h *harness) state(slug string) session.State {
	h.app.mu.Lock()
	defer h.app.mu.Unlock()
	return h.app.runners[slug].machine.State()
}

func TestFullSessionPreAndPost(t *testing.T) {
	h := newHarness(t, "", nil)
	h.tick(0)
	h.fake.SetRunning("game.exe")
	h.tick(3)
	writeFile(t, filepath.Join(h.saves, "a.sav"), "v2")
	h.fake.SetRunning()
	for i := 0; i < 6; i++ {
		h.tick(3)
	}
	if s := h.state("test-game"); s != session.WaitingStable {
		t.Fatalf("state = %v, want WaitingStable", s)
	}
	h.tick(2)
	h.tick(2)
	h.tick(0)
	if got := h.kinds(t); !slices.Equal(got, []backup.Kind{backup.KindPost, backup.KindPre}) {
		t.Fatalf("archives = %v", got)
	}
	if n := h.countMessages("Backed up Test Game"); n != 2 {
		t.Errorf("Backed up messages = %d (%v)", n, h.rec.Messages())
	}
	if s := h.state("test-game"); s != session.Idle {
		t.Errorf("state = %v, want Idle", s)
	}
}

func TestAlreadyRunningAtStartNoPre(t *testing.T) {
	h := newHarness(t, "", func(h *harness) { h.fake.SetRunning("game.exe") })
	h.tick(0)
	h.tick(3)
	if got := h.kinds(t); len(got) != 0 {
		t.Fatalf("archives = %v, want none", got)
	}
	if st := h.app.Status(); !slices.Equal(st.Running, []string{"Test Game"}) || st.Watching != 1 {
		t.Errorf("status = %+v", st)
	}
}

func TestListerErrorDoesNotEndSession(t *testing.T) {
	h := newHarness(t, "", nil)
	h.fake.SetRunning("game.exe")
	h.tick(0)
	h.fake.SetErr(errors.New("access denied"))
	for i := 0; i < 10; i++ {
		h.tick(3)
	}
	if s := h.state("test-game"); s != session.Running {
		t.Fatalf("state = %v, want Running", s)
	}
	if got := h.kinds(t); len(got) != 1 {
		t.Errorf("archives = %v, want only pre", got)
	}
}

func TestRootUnavailableNotifiesCritical(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file.txt")
	writeFile(t, file, "x")
	h := newHarness(t, filepath.Join(file, "backups"), nil)
	h.fake.SetRunning("game.exe")
	h.tick(0)
	h.tick(3)
	if n := h.countMessages("Backup failed: Test Game"); n != 1 {
		t.Fatalf("messages = %v", h.rec.Messages())
	}
	h.tick(3) // keeps ticking without panic
}

func TestHotReload(t *testing.T) {
	h := newHarness(t, "", nil)
	h.tick(0)
	other := t.TempDir()
	writeFile(t, h.cfg, configYAML(h.root, gameYAML("Test Game", "game.exe", h.saves), gameYAML("Other", "other.exe", other)))
	h.tick(3)
	if w := h.app.Status().Watching; w != 2 {
		t.Fatalf("Watching = %d, want 2", w)
	}
	writeFile(t, h.cfg, "games: [")
	h.tick(3)
	if w := h.app.Status().Watching; w != 2 {
		t.Errorf("Watching = %d after bad config, want 2", w)
	}
	if h.countMessages("Config error") != 1 {
		t.Errorf("messages = %v", h.rec.Messages())
	}
}

func TestRemovedGameFinishesSession(t *testing.T) {
	h := newHarness(t, "", nil)
	h.fake.SetRunning("game.exe")
	h.tick(0)
	writeFile(t, h.cfg, configYAML(h.root, gameYAML("Other", "other.exe", t.TempDir())))
	h.tick(3)
	if st := h.app.Status(); st.Watching != 1 {
		t.Fatalf("status = %+v", st)
	}
	writeFile(t, filepath.Join(h.saves, "a.sav"), "v2")
	h.fake.SetRunning()
	for i := 0; i < 12; i++ {
		h.tick(2)
	}
	if got := h.kinds(t); !slices.Equal(got, []backup.Kind{backup.KindPost, backup.KindPre}) {
		t.Fatalf("archives = %v, want post after removal", got)
	}
	h.app.mu.Lock()
	_, still := h.app.runners["test-game"]
	h.app.mu.Unlock()
	if still {
		t.Error("removed runner should be dropped once idle")
	}
}

func TestNextDelay(t *testing.T) {
	h := newHarness(t, "", nil)
	if d := h.app.NextDelay(); d != 3*time.Second {
		t.Fatalf("NextDelay = %v", d)
	}
	h.fake.SetRunning("game.exe")
	h.tick(0)
	h.fake.SetRunning()
	h.tick(3)
	h.tick(15)
	if d := h.app.NextDelay(); d != session.StableInterval {
		t.Errorf("NextDelay = %v, want %v", d, session.StableInterval)
	}
}

func TestBackupNow(t *testing.T) {
	h := newHarness(t, "", nil)
	h.app.BackupNow("test-game")
	h.app.Wait()
	h.tick(0)
	if got := h.kinds(t); !slices.Equal(got, []backup.Kind{backup.KindManual}) {
		t.Fatalf("archives = %v", got)
	}
}
