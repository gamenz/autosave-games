package backup

import (
	"path/filepath"
	"slices"
	"testing"

	"autosave-games/internal/config"
)

func TestCollectDirWithExclude(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.sav"), "a")
	writeFile(t, filepath.Join(dir, "sub", "b.sav"), "b")
	writeFile(t, filepath.Join(dir, "sub", "x.bak"), "x")
	got, errs := Collect([]config.SaveBase{{Index: 0, AbsBase: dir}}, []string{"*.bak"})
	if len(errs) != 0 {
		t.Fatalf("errs = %v", errs)
	}
	if want := []string{"a.sav", "sub/b.sav"}; !slices.Equal(relPaths(got), want) {
		t.Fatalf("RelPaths = %v, want %v", relPaths(got), want)
	}
	if got[0].Size != 1 || got[0].AbsPath != filepath.Join(dir, "a.sav") {
		t.Errorf("entry 0 = %+v", got[0])
	}
}

func TestCollectGlobPattern(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"user1.dat", "user2.dat", "other.dat"} {
		writeFile(t, filepath.Join(dir, n), n)
	}
	got, errs := Collect([]config.SaveBase{{AbsBase: dir, Pattern: "user*.dat"}}, nil)
	if want := []string{"user1.dat", "user2.dat"}; len(errs) != 0 || !slices.Equal(relPaths(got), want) {
		t.Fatalf("got %v (errs %v), want %v", relPaths(got), errs, want)
	}
}

func TestCollectGlobDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "slot1", "data", "s.bin"), "1")
	writeFile(t, filepath.Join(dir, "slot2", "data", "s.bin"), "2")
	got, _ := Collect([]config.SaveBase{{AbsBase: dir, Pattern: `slot?\data`}}, nil)
	if want := []string{"slot1/data/s.bin", "slot2/data/s.bin"}; !slices.Equal(relPaths(got), want) {
		t.Fatalf("got %v, want %v", relPaths(got), want)
	}
}

func TestCollectNonASCII(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Quang Nguyễn", "Saves")
	writeFile(t, filepath.Join(dir, "lưu game 1.sav"), "v")
	got, errs := Collect([]config.SaveBase{{AbsBase: dir}}, nil)
	if len(errs) != 0 || len(got) != 1 || got[0].RelPath != "lưu game 1.sav" {
		t.Fatalf("got %v, errs %v", relPaths(got), errs)
	}
}

func TestCollectMissingBaseContinues(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.sav"), "a")
	bases := []config.SaveBase{
		{Index: 0, AbsBase: filepath.Join(dir, "missing")},
		{Index: 1, AbsBase: dir},
	}
	got, errs := Collect(bases, nil)
	if len(errs) != 1 || len(got) != 1 || got[0].BaseIndex != 1 || got[0].RelPath != "a.sav" {
		t.Fatalf("got %+v, errs %v", got, errs)
	}
}

func TestCollectSingleFileBase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "profile.sav"), "p")
	got, _ := Collect([]config.SaveBase{{AbsBase: filepath.Join(dir, "profile.sav")}}, nil)
	if len(got) != 1 || got[0].RelPath != "profile.sav" {
		t.Fatalf("got %+v", got)
	}
}
