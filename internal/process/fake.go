package process

import (
	"strings"
	"sync"
)

// Fake is a test Lister with a settable process set and error.
type Fake struct {
	mu    sync.Mutex
	names map[string]bool
	err   error
}

// NewFake returns a Fake with no running processes.
func NewFake() *Fake { return &Fake{names: map[string]bool{}} }

// SetRunning replaces the running set; no arguments means nothing runs.
func (f *Fake) SetRunning(names ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.names = map[string]bool{}
	for _, n := range names {
		f.names[strings.ToLower(n)] = true
	}
}

// SetErr makes Names fail with err (nil clears it).
func (f *Fake) SetErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

// Names implements Lister.
func (f *Fake) Names() (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[string]bool, len(f.names))
	for k := range f.names {
		out[k] = true
	}
	return out, nil
}
