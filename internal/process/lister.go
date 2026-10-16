// Package process lists running process image names.
package process

import "strings"

// Lister returns the set of running process image names, lower-cased.
type Lister interface {
	Names() (map[string]bool, error)
}

// AnyRunning reports whether any of processes appears in names.
func AnyRunning(names map[string]bool, processes []string) bool {
	for _, p := range processes {
		if names[strings.ToLower(p)] {
			return true
		}
	}
	return false
}
