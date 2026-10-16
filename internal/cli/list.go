package cli

import (
	"fmt"

	"autosave-games/internal/backup"
)

func (c *command) list(args []string) int {
	if len(args) > 1 {
		return c.failf(exitUsage, "usage: autosave list [game]")
	}
	if len(args) == 1 {
		return c.listGame(args[0])
	}
	cfg, err := c.loadConfig()
	if err != nil {
		return c.failf(exitFailure, "%v", err)
	}
	svc := c.service(cfg)
	for _, g := range cfg.Games {
		archives, err := backup.List(svc.GameDir(g))
		if err != nil {
			return c.failf(exitFailure, "%v", err)
		}
		latest := "-"
		if len(archives) > 0 {
			latest = archives[0].Name
		}
		fmt.Fprintf(c.env.Stdout, "%s  (%s)  %d backups  latest %s\n", g.Name, g.Slug, len(archives), latest)
	}
	return exitOK
}

func (c *command) listGame(query string) int {
	cfg, g, code := c.loadGame(query)
	if code != exitOK {
		return code
	}
	archives, err := backup.List(c.service(cfg).GameDir(*g))
	if err != nil {
		return c.failf(exitFailure, "%v", err)
	}
	for _, a := range archives {
		fmt.Fprintf(c.env.Stdout, "%s  %s  %d KB\n", a.Name, a.Kind, (a.Size+1023)/1024)
	}
	return exitOK
}
