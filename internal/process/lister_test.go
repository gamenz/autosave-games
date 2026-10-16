package process

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSnapshotRunning(t *testing.T) {
	s := Snapshot{Names: map[string]bool{"eldenring.exe": true},
		Paths: []string{`d:\steam\steamapps\common\elden ring\game\eldenring.exe`}}
	cases := []struct {
		procs []string
		dir   string
		want  bool
	}{
		{[]string{"EldenRing.exe"}, "", true},
		{nil, `D:\Steam\steamapps\common\ELDEN RING`, true},
		{nil, `D:\Steam\steamapps\common\ELDEN`, false},
		{[]string{"hk.exe"}, `D:\Steam\steamapps\common\Hollow Knight`, false},
		{nil, "", false},
	}
	for _, tc := range cases {
		if got := s.Running(tc.procs, tc.dir); got != tc.want {
			t.Errorf("Running(%v, %q) = %v, want %v", tc.procs, tc.dir, got, tc.want)
		}
	}
}

func TestSystemListerSeesSelf(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	snap, err := NewSystemLister().Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if self := strings.ToLower(filepath.Base(exe)); !snap.Names[self] {
		t.Errorf("own process %q not in names", self)
	}
	if !slices.Contains(snap.Paths, strings.ToLower(exe)) {
		t.Errorf("own path %q not in %d paths", exe, len(snap.Paths))
	}
}

func TestFake(t *testing.T) {
	f := NewFake()
	f.SetRunning("Game.exe")
	snap, err := f.Snapshot()
	if err != nil || !snap.Names["game.exe"] || len(snap.Paths) != 0 {
		t.Fatalf("Snapshot = %+v, %v", snap, err)
	}
	f.SetRunningPaths(`D:\G\X.exe`)
	snap, _ = f.Snapshot()
	if !slices.Equal(snap.Paths, []string{`d:\g\x.exe`}) || !snap.Names["x.exe"] {
		t.Fatalf("Snapshot after paths = %+v", snap)
	}
	want := errors.New("boom")
	f.SetErr(want)
	if _, err := f.Snapshot(); !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
}

func BenchmarkSystemSnapshot(b *testing.B) {
	l := NewSystemLister()
	for i := 0; i < b.N; i++ {
		if _, err := l.Snapshot(); err != nil {
			b.Fatal(err)
		}
	}
}
