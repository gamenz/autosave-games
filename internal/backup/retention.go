package backup

import (
	"fmt"
	"os"
)

// Prune deletes the oldest archives so that at most keepLast non-prerestore
// archives remain. Prerestore archives are never pruned.
func Prune(dir string, keepLast int) (removed []string, err error) {
	archives, err := List(dir)
	if err != nil {
		return nil, err
	}
	kept := 0
	for _, a := range archives {
		if a.Kind == KindPreRestore {
			continue
		}
		kept++
		if kept <= keepLast {
			continue
		}
		if err := os.Remove(a.Path); err != nil {
			return removed, fmt.Errorf("prune %s: %w", a.Name, err)
		}
		removed = append(removed, a.Path)
	}
	return removed, nil
}
