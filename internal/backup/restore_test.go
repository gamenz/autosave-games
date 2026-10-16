package backup

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRestoreRoundTrip(t *testing.T) {
	s, g, saves := newTestService(t)
	uni := filepath.Join(saves, "sub", "lưu game 1.sav")
	writeFile(t, uni, "orig")
	r := mustRun(t, s, g, KindManual)

	writeFile(t, filepath.Join(saves, "a.sav"), "corrupted")
	os.Remove(uni)
	writeFile(t, filepath.Join(saves, "new.sav"), "keep me")

	written, err := Restore(r.Path)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if len(written) != 2 {
		t.Errorf("written = %v", written)
	}
	if got := readFile(t, filepath.Join(saves, "a.sav")); got != "v1" {
		t.Errorf("a.sav = %q", got)
	}
	if got := readFile(t, uni); got != "orig" {
		t.Errorf("unicode file = %q", got)
	}
	if got := readFile(t, filepath.Join(saves, "new.sav")); got != "keep me" {
		t.Errorf("new.sav = %q (must be untouched)", got)
	}
}

func TestRestoreRecreatesMissingBase(t *testing.T) {
	s, g, saves := newTestService(t)
	r := mustRun(t, s, g, KindManual)
	if err := os.RemoveAll(saves); err != nil {
		t.Fatal(err)
	}
	if _, err := Restore(r.Path); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(saves, "a.sav")); got != "v1" {
		t.Errorf("a.sav = %q", got)
	}
}

func handZip(t *testing.T, base string, entryName string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "hand.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create(entryName)
	w.Write([]byte("evil"))
	mw, _ := zw.Create(manifestName)
	json.NewEncoder(mw).Encode(Manifest{Game: "X", Kind: KindManual, Bases: []BaseRef{{Index: 0, AbsPath: base}}})
	zw.Close()
	f.Close()
	return p
}

func TestRestoreRejectsZipSlip(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "a", "b")
	p := handZip(t, base, "files/0/../../evil.txt")
	if _, err := Restore(p); err == nil {
		t.Fatal("expected zip slip error")
	}
	if _, err := os.Stat(filepath.Join(root, "evil.txt")); err == nil {
		t.Fatal("evil.txt was written")
	}
}

func TestRestoreUnknownBaseIndex(t *testing.T) {
	p := handZip(t, t.TempDir(), "files/7/a.sav")
	if _, err := Restore(p); err == nil {
		t.Fatal("expected unknown base error")
	}
}

func TestPlanRestore(t *testing.T) {
	s, g, saves := newTestService(t)
	r := mustRun(t, s, g, KindManual)
	targets, err := PlanRestore(r.Path)
	if err != nil || len(targets) != 1 || targets[0].AbsPath != filepath.Join(saves, "a.sav") {
		t.Fatalf("PlanRestore = %+v, %v", targets, err)
	}
}

func TestFindArchive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "2026-10-17_21-30-05_pre.zip"), "")
	writeFile(t, filepath.Join(dir, "2026-10-17_22-00-00_post.zip"), "")
	cases := []struct{ prefix, want, wantErr string }{
		{"", "2026-10-17_22-00-00_post.zip", ""},
		{"2026-10-17_21", "2026-10-17_21-30-05_pre.zip", ""},
		{"2026-10-17", "", "ambiguous"},
		{"2025", "", "no archive matches"},
		{"2026-10-17_23", "2026-10-17_23-00-00_prerestore.zip", ""},
	}
	writeFile(t, filepath.Join(dir, "2026-10-17_23-00-00_prerestore.zip"), "")
	for _, tc := range cases {
		got, err := FindArchive(dir, tc.prefix)
		if tc.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("FindArchive(%q) err = %v, want %q", tc.prefix, err, tc.wantErr)
			}
			continue
		}
		if err != nil || got.Name != tc.want {
			t.Errorf("FindArchive(%q) = %q, %v", tc.prefix, got.Name, err)
		}
	}
}
