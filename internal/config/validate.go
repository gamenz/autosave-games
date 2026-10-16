package config

import (
	"errors"
	"fmt"
	"strings"
)

// Validate reports every problem in c, joined with errors.Join.
func Validate(c *Config) error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	if strings.TrimSpace(c.BackupRoot) == "" {
		add("backup_root: required")
	}
	if c.PollSeconds < 0 {
		add("poll_seconds: must be > 0")
	}
	switch c.Notify {
	case NotifyAll, NotifyErrors, NotifyNone:
	default:
		add("notify: must be one of all, errors, none")
	}
	if c.Defaults.KeepLast < 0 {
		add("defaults.keep_last: must be > 0")
	}
	if c.Defaults.SettleSeconds < 0 {
		add("defaults.settle_seconds: must be > 0")
	}

	names := map[string]bool{}
	slugs := map[string]bool{}
	for i, g := range c.Games {
		label := gameLabel(i, g)
		if strings.TrimSpace(g.Name) == "" {
			add("%s: name: required", label)
		} else {
			key := strings.ToLower(strings.TrimSpace(g.Name))
			if names[key] {
				add("%s: duplicate name", label)
			}
			names[key] = true
			switch {
			case g.Slug == "":
				add("%s: name: no letters or digits", label)
			case slugs[g.Slug]:
				add("%s: duplicate slug %q", label, g.Slug)
			}
			slugs[g.Slug] = true
		}
		if len(g.Processes) == 0 {
			add("%s: processes: at least one required", label)
		}
		if len(g.Saves) == 0 {
			add("%s: saves: at least one required", label)
		}
		if g.KeepLast < 0 {
			add("%s: keep_last: must be > 0", label)
		}
		if g.SettleSeconds < 0 {
			add("%s: settle_seconds: must be > 0", label)
		}
	}
	return errors.Join(errs...)
}

func gameLabel(index int, g Game) string {
	if strings.TrimSpace(g.Name) == "" {
		return fmt.Sprintf("game #%d", index+1)
	}
	return fmt.Sprintf("game %q", g.Name)
}
