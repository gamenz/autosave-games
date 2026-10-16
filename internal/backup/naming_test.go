package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var ts = time.Date(2026, 10, 17, 21, 30, 5, 0, time.Local)

func TestArchiveName(t *testing.T) {
	if got := ArchiveName(ts, KindPre, 1); got != "2026-10-17_21-30-05_pre.zip" {
		t.Errorf("ArchiveName = %q", got)
	}
	if got := ArchiveName(ts, KindManual, 2); got != "2026-10-17_21-30-05_manual_2.zip" {
		t.Errorf("ArchiveName n=2 = %q", got)
	}
	for _, tc := range []struct {
		name string
		kind Kind
	}{{"2026-10-17_21-30-05_pre.zip", KindPre}, {"2026-10-17_21-30-05_manual_2.zip", KindManual},
		{"2026-10-17_21-30-05_prerestore.zip", KindPreRestore}} {
		got, kind, ok := ParseArchiveName(tc.name)
		if !ok || !got.Equal(ts) || kind != tc.kind {
			t.Errorf("ParseArchiveName(%q) = %v %q %v", tc.name, got, kind, ok)
		}
	}
	for _, bad := range []string{"notes.zip", "2026-10-17_21-30-05_pre.zip.tmp", "2026-10-17_21-30-05_weird.zip", "2026-10-17_21-30-05_pre_x.zip"} {
		if _, _, ok := ParseArchiveName(bad); ok {
			t.Errorf("ParseArchiveName(%q) should fail", bad)
		}
	}
}

func TestUniqueArchivePath(t *testing.T) {
	dir := t.TempDir()
	for _, want := range []string{"2026-10-17_21-30-05_pre.zip", "2026-10-17_21-30-05_pre_2.zip", "2026-10-17_21-30-05_pre_3.zip"} {
		got := UniqueArchivePath(dir, ts, KindPre)
		if got != filepath.Join(dir, want) {
			t.Fatalf("UniqueArchivePath = %q, want %q", got, want)
		}
		if err := os.WriteFile(got, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
