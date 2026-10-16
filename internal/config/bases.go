package config

import (
	"os"
	"path/filepath"
	"strings"
)

// SaveBase is one resolved "saves" entry: a literal directory or file path
// plus an optional glob pattern relative to it ("" means the base itself).
type SaveBase struct {
	Index   int
	AbsBase string
	Pattern string
}

// SplitGlobBase splits p at the first path segment containing a glob
// character (*, ?, [) into a literal base and the remaining pattern.
func SplitGlobBase(p string) (base, pattern string) {
	sep := string(os.PathSeparator)
	segments := strings.Split(filepath.Clean(p), sep)
	for i, seg := range segments {
		if strings.ContainsAny(seg, "*?[") {
			base = strings.Join(segments[:i], sep)
			if base == "" {
				base = "."
			} else if strings.HasSuffix(base, ":") {
				base += sep
			}
			return base, strings.Join(segments[i:], sep)
		}
	}
	return filepath.Clean(p), ""
}

// ResolveSaves turns expanded save paths into indexed save bases.
func ResolveSaves(saves []string) []SaveBase {
	bases := make([]SaveBase, 0, len(saves))
	for i, s := range saves {
		base, pattern := SplitGlobBase(s)
		bases = append(bases, SaveBase{Index: i, AbsBase: base, Pattern: pattern})
	}
	return bases
}

// IsWithin reports whether child equals parent or lies under it.
// Comparison is case-insensitive and segment-aware (Windows semantics).
func IsWithin(child, parent string) bool {
	c := strings.ToLower(filepath.Clean(child))
	p := strings.ToLower(filepath.Clean(parent))
	if c == p {
		return true
	}
	if !strings.HasSuffix(p, string(os.PathSeparator)) {
		p += string(os.PathSeparator)
	}
	return strings.HasPrefix(c, p)
}
