// Package process lists running processes (image names and exe paths).
package process

import (
	"os"
	"path/filepath"
	"strings"
)

// Snapshot is one listing of running processes. Names and Paths are
// lower-cased; processes whose exe path cannot be read appear in Names only.
type Snapshot struct {
	Names map[string]bool
	Paths []string
}

// Lister takes process snapshots.
type Lister interface {
	Snapshot() (Snapshot, error)
}

// Running reports whether any of processes is running, or any running exe
// lies inside installDir (empty installDir disables the path check).
func (s Snapshot) Running(processes []string, installDir string) bool {
	for _, p := range processes {
		if s.Names[strings.ToLower(p)] {
			return true
		}
	}
	if strings.TrimSpace(installDir) == "" {
		return false
	}
	dir := strings.ToLower(filepath.Clean(installDir))
	if !strings.HasSuffix(dir, string(os.PathSeparator)) {
		dir += string(os.PathSeparator)
	}
	for _, p := range s.Paths {
		if strings.HasPrefix(p, dir) {
			return true
		}
	}
	return false
}
