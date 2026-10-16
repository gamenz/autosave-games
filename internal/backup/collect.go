// Package backup collects save files, writes zip archives and restores them.
package backup

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"

	"autosave-games/internal/config"
)

// Entry is one save file to archive.
type Entry struct {
	BaseIndex int
	RelPath   string // forward slashes, relative to the save base
	AbsPath   string
	Size      int64
}

// NormalizeBases rewrites a base that points at a single file into its
// parent directory plus the file name as pattern, so relative paths and
// restore targets are always computed from a directory.
func NormalizeBases(bases []config.SaveBase) []config.SaveBase {
	out := make([]config.SaveBase, len(bases))
	for i, b := range bases {
		out[i] = b
		if b.Pattern != "" {
			continue
		}
		if info, err := os.Stat(b.AbsBase); err == nil && !info.IsDir() {
			out[i].AbsBase, out[i].Pattern = filepath.Dir(b.AbsBase), filepath.Base(b.AbsBase)
		}
	}
	return out
}

// Collect lists the files of every save base, applying exclude patterns.
// Problems with one base are reported and the others still collected.
func Collect(bases []config.SaveBase, exclude []string) ([]Entry, []error) {
	var entries []Entry
	var errs []error
	for _, b := range NormalizeBases(bases) {
		found, err := collectBase(b)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, e := range found {
			if !excluded(e.RelPath, exclude) {
				entries = append(entries, e)
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].BaseIndex != entries[j].BaseIndex {
			return entries[i].BaseIndex < entries[j].BaseIndex
		}
		return entries[i].RelPath < entries[j].RelPath
	})
	return entries, errs
}

// Paths returns the absolute path of every entry.
func Paths(entries []Entry) []string {
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.AbsPath
	}
	return out
}

func collectBase(b config.SaveBase) ([]Entry, error) {
	matches := []string{b.AbsBase}
	if b.Pattern != "" {
		var err error
		matches, err = filepath.Glob(filepath.Join(b.AbsBase, b.Pattern))
		if err != nil {
			return nil, fmt.Errorf("save %d: bad pattern %q: %w", b.Index, b.Pattern, err)
		}
	}
	var entries []Entry
	for _, m := range matches {
		found, err := walk(b, m)
		if err != nil {
			return nil, fmt.Errorf("save %d: %w", b.Index, err)
		}
		entries = append(entries, found...)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("save %d: no files found in %s", b.Index, filepath.Join(b.AbsBase, b.Pattern))
	}
	return entries, nil
}

func walk(b config.SaveBase, root string) ([]Entry, error) {
	var entries []Entry
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(b.AbsBase, p)
		if err != nil {
			return err
		}
		entries = append(entries, Entry{
			BaseIndex: b.Index, RelPath: filepath.ToSlash(rel), AbsPath: p, Size: info.Size(),
		})
		return nil
	})
	return entries, err
}

func excluded(rel string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := path.Match(p, rel); ok {
			return true
		}
		if ok, _ := path.Match(p, path.Base(rel)); ok {
			return true
		}
	}
	return false
}
