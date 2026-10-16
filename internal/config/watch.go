package config

import (
	"os"
	"time"
)

// FileWatcher detects edits of the config file by mtime and size.
type FileWatcher struct {
	path    string
	modTime time.Time
	size    int64
	seen    bool
}

// NewFileWatcher watches path; call Changed once to record the initial state.
func NewFileWatcher(path string) *FileWatcher { return &FileWatcher{path: path} }

// Changed reports whether the file differs from the previous call. The first
// call only records the state. A missing file counts as unchanged.
func (w *FileWatcher) Changed() bool {
	info, err := os.Stat(w.path)
	if err != nil {
		return false
	}
	changed := w.seen && (!info.ModTime().Equal(w.modTime) || info.Size() != w.size)
	w.modTime, w.size, w.seen = info.ModTime(), info.Size(), true
	return changed
}
