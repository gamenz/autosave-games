package backup

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Kind tells why an archive was created.
type Kind string

const (
	KindPre        Kind = "pre"
	KindPost       Kind = "post"
	KindManual     Kind = "manual"
	KindPreRestore Kind = "prerestore"
)

const timeLayout = "2006-01-02_15-04-05"

func validKind(k Kind) bool {
	switch k {
	case KindPre, KindPost, KindManual, KindPreRestore:
		return true
	}
	return false
}

// ArchiveName formats "<time>_<kind>.zip", or "<time>_<kind>_<n>.zip" for n > 1.
func ArchiveName(t time.Time, kind Kind, n int) string {
	name := t.Format(timeLayout) + "_" + string(kind)
	if n > 1 {
		name += "_" + strconv.Itoa(n)
	}
	return name + ".zip"
}

// ParseArchiveName is the inverse of ArchiveName (local time).
func ParseArchiveName(name string) (time.Time, Kind, bool) {
	stem, ok := strings.CutSuffix(name, ".zip")
	if !ok || len(stem) < len(timeLayout)+2 || stem[len(timeLayout)] != '_' {
		return time.Time{}, "", false
	}
	t, err := time.ParseInLocation(timeLayout, stem[:len(timeLayout)], time.Local)
	if err != nil {
		return time.Time{}, "", false
	}
	rest := stem[len(timeLayout)+1:]
	kindPart, seq, hasSeq := strings.Cut(rest, "_")
	if hasSeq {
		if n, err := strconv.Atoi(seq); err != nil || n < 2 {
			return time.Time{}, "", false
		}
	}
	kind := Kind(kindPart)
	if !validKind(kind) {
		return time.Time{}, "", false
	}
	return t, kind, true
}

// UniqueArchivePath returns the first archive path in dir that does not
// exist yet, so two backups in the same second never overwrite each other.
func UniqueArchivePath(dir string, t time.Time, kind Kind) string {
	for n := 1; ; n++ {
		p := filepath.Join(dir, ArchiveName(t, kind, n))
		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			return p
		}
	}
}
