package backup

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// LockStaleAfter is how old a .lock file must be before it is taken over.
const LockStaleAfter = 10 * time.Minute

// ErrLocked means another backup of the same game is in progress.
var ErrLocked = errors.New("backup already in progress")

// AcquireLock creates dir/.lock exclusively. The returned release func
// removes it.
func AcquireLock(dir string, now time.Time) (release func(), err error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}
	lock := filepath.Join(dir, ".lock")
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(f, "pid %d at %s\n", os.Getpid(), now.Format(time.RFC3339))
			f.Close()
			return func() { os.Remove(lock) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("create lock: %w", err)
		}
		info, statErr := os.Stat(lock)
		if statErr != nil || now.Sub(info.ModTime()) <= LockStaleAfter {
			return nil, ErrLocked
		}
		os.Remove(lock)
	}
	return nil, ErrLocked
}
