package config

import (
	"errors"
	"fmt"
	"strings"
)

type errorList []error

func (l *errorList) add(format string, args ...any) {
	*l = append(*l, fmt.Errorf(format, args...))
}

// Validate reports every problem in c, joined with errors.Join.
func Validate(c *Config) error {
	var errs errorList
	if strings.TrimSpace(c.BackupRoot) == "" {
		errs.add("backup_root: required")
	}
	if c.PollSeconds < 0 {
		errs.add("poll_seconds: must be > 0")
	}
	switch c.Notify {
	case NotifyAll, NotifyErrors, NotifyNone:
	default:
		errs.add("notify: must be one of all, errors, none")
	}
	if c.Defaults.KeepLast < 0 {
		errs.add("defaults.keep_last: must be > 0")
	}
	if c.Defaults.SettleSeconds < 0 {
		errs.add("defaults.settle_seconds: must be > 0")
	}
	names := map[string]bool{}
	slugs := map[string]bool{}
	for i, g := range c.Games {
		validateGame(&errs, gameLabel(i, g), g, c.BackupRoot, names, slugs)
	}
	return errors.Join(errs...)
}

func validateGame(errs *errorList, label string, g Game, root string, names, slugs map[string]bool) {
	if strings.TrimSpace(g.Name) == "" {
		errs.add("%s: name: required", label)
	} else {
		key := strings.ToLower(strings.TrimSpace(g.Name))
		if names[key] {
			errs.add("%s: duplicate name", label)
		}
		names[key] = true
		switch {
		case g.Slug == "":
			errs.add("%s: name: no letters or digits", label)
		case slugs[g.Slug]:
			errs.add("%s: duplicate slug %q", label, g.Slug)
		}
		slugs[g.Slug] = true
	}
	if len(g.Processes) == 0 {
		errs.add("%s: processes: at least one required", label)
	}
	if len(g.Saves) == 0 {
		errs.add("%s: saves: at least one required", label)
	}
	if g.KeepLast < 0 {
		errs.add("%s: keep_last: must be > 0", label)
	}
	if g.SettleSeconds < 0 {
		errs.add("%s: settle_seconds: must be > 0", label)
	}
	if root = strings.TrimSpace(root); root != "" {
		for _, b := range ResolveSaves(g.Saves) {
			if IsWithin(b.AbsBase, root) || IsWithin(root, b.AbsBase) {
				errs.add("%s: saves[%d]: overlaps backup_root", label, b.Index)
			}
		}
	}
}

func gameLabel(index int, g Game) string {
	if strings.TrimSpace(g.Name) == "" {
		return fmt.Sprintf("game #%d", index+1)
	}
	return fmt.Sprintf("game %q", g.Name)
}
