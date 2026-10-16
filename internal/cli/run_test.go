package cli

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"autosave-games/internal/backup"
	"autosave-games/internal/process"
)

type fixture struct {
	cfg, root, saves string
	fake             *process.Fake
	stdout, stderr   *bytes.Buffer
	stdin            string
	clock            time.Time
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{root: t.TempDir(), saves: t.TempDir(), fake: process.NewFake(),
		clock: time.Date(2026, 10, 17, 21, 0, 0, 0, time.Local)}
	f.cfg = filepath.Join(t.TempDir(), "games.yaml")
	writeFile(t, filepath.Join(f.saves, "a.sav"), "v1")
	writeFile(t, f.cfg, fmt.Sprintf("backup_root: '%s'\ngames:\n  - {name: Test Game, processes: [game.exe], saves: ['%s']}\n", f.root, f.saves))
	return f
}

func (f *fixture) run(args ...string) int {
	f.stdout, f.stderr = &bytes.Buffer{}, &bytes.Buffer{}
	e := Env{Stdout: f.stdout, Stderr: f.stderr, Stdin: strings.NewReader(f.stdin), Lister: f.fake,
		Log: slog.New(slog.DiscardHandler), Version: "test",
		Now: func() time.Time { f.clock = f.clock.Add(time.Minute); return f.clock }}
	return Run(append([]string{"--config", f.cfg}, args...), e)
}

func (f *fixture) archives(t *testing.T) []backup.ArchiveInfo {
	t.Helper()
	list, err := backup.List(filepath.Join(f.root, "test-game"))
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestListAndBackup(t *testing.T) {
	f := newFixture(t)
	if code := f.run("backup", "test game"); code != 0 {
		t.Fatalf("backup exit %d: %s", code, f.stderr)
	}
	if n := len(f.archives(t)); n != 1 {
		t.Fatalf("archives = %d", n)
	}
	if code := f.run("list"); code != 0 || !strings.Contains(f.stdout.String(), "Test Game  (test-game)  1 backups") {
		t.Errorf("list exit %d: %q", code, f.stdout)
	}
	if code := f.run("list", "test-game"); code != 0 || !strings.Contains(f.stdout.String(), "_manual.zip  manual") {
		t.Errorf("list game exit %d: %q", code, f.stdout)
	}
}

func TestUnknownGame(t *testing.T) {
	f := newFixture(t)
	code := f.run("backup", "zelda")
	if code != 2 || !strings.Contains(f.stderr.String(), `unknown game "zelda"; known: test-game`) {
		t.Errorf("exit %d, stderr %q", code, f.stderr)
	}
}

func TestUsage(t *testing.T) {
	f := newFixture(t)
	if code := f.run(); code != 2 {
		t.Errorf("no args exit %d", code)
	}
	if code := f.run("explode"); code != 2 {
		t.Errorf("bad subcommand exit %d", code)
	}
	if code := f.run("backup"); code != 2 {
		t.Errorf("missing game exit %d", code)
	}
}

func TestValidate(t *testing.T) {
	f := newFixture(t)
	if code := f.run("validate"); code != 0 || !strings.HasPrefix(f.stdout.String(), "OK") ||
		!strings.Contains(f.stdout.String(), "Test Game: "+f.saves) {
		t.Errorf("validate exit %d: %q", code, f.stdout)
	}
	writeFile(t, f.cfg, "games: [")
	if code := f.run("validate"); code != 1 {
		t.Errorf("broken config exit %d", code)
	}
}
