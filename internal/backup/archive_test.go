package backup

import (
	"archive/zip"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"autosave-games/internal/config"
)

func TestWriteArchiveRoundTrip(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "a.sav"), "alpha")
	writeFile(t, filepath.Join(src, "lưu game 1.sav"), "beta")
	entries, errs := Collect([]config.SaveBase{{AbsBase: src}}, nil)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	m := Manifest{Game: "Test", Kind: KindPre, CreatedAt: ts, AppVersion: "test", Hash: "abc",
		Bases: []BaseRef{{Index: 0, AbsPath: src}}}
	dst := filepath.Join(t.TempDir(), "out.zip")
	if err := WriteArchive(dst, entries, m); err != nil {
		t.Fatalf("WriteArchive: %v", err)
	}

	r, err := zip.OpenReader(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var names []string
	contents := map[string]string{}
	for _, f := range r.File {
		names = append(names, f.Name)
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		contents[f.Name] = string(b)
	}
	if want := []string{"files/0/a.sav", "files/0/lưu game 1.sav", "manifest.json"}; !slices.Equal(names, want) {
		t.Fatalf("zip names = %v, want %v", names, want)
	}
	if contents["files/0/lưu game 1.sav"] != "beta" {
		t.Errorf("content = %q", contents["files/0/lưu game 1.sav"])
	}
	got, err := ReadManifest(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CreatedAt.Equal(m.CreatedAt) {
		t.Errorf("CreatedAt = %v", got.CreatedAt)
	}
	got.CreatedAt = m.CreatedAt
	if !reflect.DeepEqual(got, m) {
		t.Errorf("manifest = %+v, want %+v", got, m)
	}
	if _, err := os.Stat(dst + ".tmp"); !errors.Is(err, fs.ErrNotExist) {
		t.Error("tmp file left behind")
	}
}

func TestWriteArchiveFailureLeavesNothing(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "out.zip")
	entries := []Entry{{RelPath: "gone.sav", AbsPath: filepath.Join(t.TempDir(), "gone.sav")}}
	if err := WriteArchive(dst, entries, Manifest{}); err == nil {
		t.Fatal("expected error")
	}
	for _, p := range []string{dst, dst + ".tmp"} {
		if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s should not exist", p)
		}
	}
}
