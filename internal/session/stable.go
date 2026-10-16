package session

import (
	"fmt"
	"os"
	"time"
)

// FileStat is the part of a file's metadata used to detect ongoing writes.
type FileStat struct {
	Size    int64
	ModTime time.Time
}

// StatFiles stats every path; any failure aborts with an error.
func StatFiles(paths []string) (map[string]FileStat, error) {
	stats := make(map[string]FileStat, len(paths))
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("stat save file: %w", err)
		}
		stats[p] = FileStat{Size: info.Size(), ModTime: info.ModTime()}
	}
	return stats, nil
}

// CanOpen reports whether path can be opened for reading right now.
// A game holding the file without read sharing makes this fail.
func CanOpen(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// Stable reports whether two consecutive observations show the same files
// with unchanged size and mtime, all currently openable.
func Stable(prev, cur map[string]FileStat, canOpen func(string) bool) bool {
	if prev == nil || len(prev) != len(cur) {
		return false
	}
	for p, c := range cur {
		old, ok := prev[p]
		if !ok || old.Size != c.Size || !old.ModTime.Equal(c.ModTime) {
			return false
		}
	}
	for p := range cur {
		if !canOpen(p) {
			return false
		}
	}
	return true
}
