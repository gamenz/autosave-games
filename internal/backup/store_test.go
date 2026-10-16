package backup

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func touchArchive(t *testing.T, dir string, sec int, kind Kind) string {
	t.Helper()
	name := ArchiveName(ts.Add(time.Duration(sec)*time.Second), kind, 1)
	writeFile(t, filepath.Join(dir, name), "")
	return name
}

func names(infos []ArchiveInfo) []string {
	out := make([]string, len(infos))
	for i, a := range infos {
		out[i] = a.Name
	}
	return out
}

func TestListOrderAndFilter(t *testing.T) {
	dir := t.TempDir()
	a := touchArchive(t, dir, 0, KindPre)
	c := touchArchive(t, dir, 20, KindManual)
	b := touchArchive(t, dir, 10, KindPost)
	writeFile(t, filepath.Join(dir, c+".tmp"), "")
	writeFile(t, filepath.Join(dir, "notes.txt"), "")
	got, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{c, b, a}; !slices.Equal(names(got), want) {
		t.Fatalf("List = %v, want %v", names(got), want)
	}
	if got[0].Kind != KindManual || got[0].Path != filepath.Join(dir, c) {
		t.Errorf("first = %+v", got[0])
	}
}

func TestListMissingDir(t *testing.T) {
	got, err := List(filepath.Join(t.TempDir(), "nope"))
	if err != nil || len(got) != 0 {
		t.Fatalf("List = %v, %v", got, err)
	}
}

func TestPruneKeepsPreRestore(t *testing.T) {
	dir := t.TempDir()
	pre := touchArchive(t, dir, 0, KindPre)
	pr := touchArchive(t, dir, 1, KindPreRestore)
	oldPost := touchArchive(t, dir, 2, KindPost)
	manual := touchArchive(t, dir, 3, KindManual)
	post := touchArchive(t, dir, 4, KindPost)
	removed, err := Prune(dir, 2)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{filepath.Join(dir, oldPost), filepath.Join(dir, pre)}; !slices.Equal(removed, want) {
		t.Errorf("removed = %v, want %v", removed, want)
	}
	left, _ := List(dir)
	if want := []string{post, manual, pr}; !slices.Equal(names(left), want) {
		t.Errorf("left = %v, want %v", names(left), want)
	}
	if _, err := os.Stat(filepath.Join(dir, pr)); err != nil {
		t.Error("prerestore archive was pruned")
	}
}
