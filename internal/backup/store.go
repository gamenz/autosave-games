package backup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ArchiveInfo describes one archive file on disk.
type ArchiveInfo struct {
	Path string
	Name string
	Time time.Time
	Kind Kind
	Size int64
}

// List returns the valid archives in dir, newest first. A missing dir is
// treated as empty.
func List(dir string) ([]ArchiveInfo, error) {
	items, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list archives: %w", err)
	}
	var out []ArchiveInfo
	for _, it := range items {
		if it.IsDir() {
			continue
		}
		t, kind, ok := ParseArchiveName(it.Name())
		if !ok {
			continue
		}
		var size int64
		if info, err := it.Info(); err == nil {
			size = info.Size()
		}
		out = append(out, ArchiveInfo{
			Path: filepath.Join(dir, it.Name()), Name: it.Name(), Time: t, Kind: kind, Size: size,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Time.Equal(out[j].Time) {
			return out[i].Time.After(out[j].Time)
		}
		return out[i].Name > out[j].Name
	})
	return out, nil
}
