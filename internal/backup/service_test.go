package backup

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"autosave-games/internal/config"
)

func newTestService(t *testing.T) (*Service, config.Game, string) {
	t.Helper()
	root, saves := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(saves, "a.sav"), "v1")
	clock := ts
	s := &Service{Root: root, AppVersion: "test", Now: func() time.Time {
		clock = clock.Add(time.Second)
		return clock
	}}
	g := config.Game{Name: "Test Game", Slug: "test-game", Saves: []string{saves}, KeepLast: 20}
	return s, g, saves
}

func mustRun(t *testing.T, s *Service, g config.Game, kind Kind) Result {
	t.Helper()
	r, err := s.Run(g, kind)
	if err != nil {
		t.Fatalf("Run(%s): %v", kind, err)
	}
	return r
}

func TestRunWritesThenDedups(t *testing.T) {
	s, g, _ := newTestService(t)
	if r := mustRun(t, s, g, KindPre); r.Skipped || filepath.Dir(r.Path) != s.GameDir(g) {
		t.Fatalf("pre = %+v", r)
	}
	if r := mustRun(t, s, g, KindPost); !r.Skipped || r.Reason != ReasonUnchanged {
		t.Fatalf("post = %+v, want skipped unchanged", r)
	}
	if r := mustRun(t, s, g, KindManual); r.Skipped {
		t.Fatalf("manual = %+v, want written", r)
	}
	got, _ := List(s.GameDir(g))
	if len(got) != 2 || got[0].Kind != KindManual || got[1].Kind != KindPre {
		t.Fatalf("archives = %v", names(got))
	}
	m, err := ReadManifest(got[0].Path)
	if err != nil || m.Game != "Test Game" || m.Kind != KindManual || m.Hash == "" || len(m.Bases) != 1 {
		t.Errorf("manifest = %+v, %v", m, err)
	}
}

func TestRunAfterChangeWritesPost(t *testing.T) {
	s, g, saves := newTestService(t)
	mustRun(t, s, g, KindPre)
	writeFile(t, filepath.Join(saves, "a.sav"), "v2")
	if r := mustRun(t, s, g, KindPost); r.Skipped {
		t.Fatalf("post = %+v, want written", r)
	}
}

func TestRunNoFiles(t *testing.T) {
	s, g, _ := newTestService(t)
	g.Saves = []string{filepath.Join(t.TempDir(), "missing")}
	r := mustRun(t, s, g, KindPost)
	if !r.Skipped || r.Reason != ReasonNoFiles || len(r.Warnings) != 1 {
		t.Fatalf("result = %+v", r)
	}
}

func TestRunRootUnavailable(t *testing.T) {
	s, g, _ := newTestService(t)
	file := filepath.Join(t.TempDir(), "file.txt")
	writeFile(t, file, "x")
	s.Root = filepath.Join(file, "backups")
	if _, err := s.Run(g, KindPre); err == nil {
		t.Fatal("expected error for unusable backup root")
	}
}

func TestRunLocked(t *testing.T) {
	s, g, _ := newTestService(t)
	release, err := AcquireLock(s.GameDir(g), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := s.Run(g, KindManual); !errors.Is(err, ErrLocked) {
		t.Fatalf("err = %v, want ErrLocked", err)
	}
}

func TestRunPrunes(t *testing.T) {
	s, g, _ := newTestService(t)
	g.KeepLast = 2
	for i := 0; i < 4; i++ {
		mustRun(t, s, g, KindManual)
	}
	if got, _ := List(s.GameDir(g)); len(got) != 2 {
		t.Fatalf("archives = %v, want 2", names(got))
	}
}
