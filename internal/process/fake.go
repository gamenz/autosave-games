package process

import (
	"path/filepath"
	"strings"
	"sync"
)

// Fake is a test Lister with a settable snapshot and error.
type Fake struct {
	mu    sync.Mutex
	names map[string]bool
	paths []string
	err   error
}

// NewFake returns a Fake with no running processes.
func NewFake() *Fake { return &Fake{names: map[string]bool{}} }

// SetRunning replaces the running set with names only (no paths);
// no arguments means nothing runs.
func (f *Fake) SetRunning(names ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.names, f.paths = map[string]bool{}, nil
	for _, n := range names {
		f.names[strings.ToLower(n)] = true
	}
}

// SetRunningPaths replaces the running set with full exe paths; each
// path's base name is added to Names too.
func (f *Fake) SetRunningPaths(paths ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.names, f.paths = map[string]bool{}, nil
	for _, p := range paths {
		p = strings.ToLower(p)
		f.paths = append(f.paths, p)
		f.names[filepath.Base(p)] = true
	}
}

// SetErr makes Snapshot fail with err (nil clears it).
func (f *Fake) SetErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

// Snapshot implements Lister.
func (f *Fake) Snapshot() (Snapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return Snapshot{}, f.err
	}
	snap := Snapshot{Names: make(map[string]bool, len(f.names)), Paths: append([]string(nil), f.paths...)}
	for k := range f.names {
		snap.Names[k] = true
	}
	return snap, nil
}
