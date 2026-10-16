package cli

import (
	"fmt"

	"autosave-games/internal/backup"
)

func (c *command) backup(args []string) int {
	if len(args) != 1 {
		return c.failf(exitUsage, "usage: autosave backup <game>")
	}
	cfg, g, code := c.loadGame(args[0])
	if code != exitOK {
		return code
	}
	res, err := c.service(cfg).Run(*g, backup.KindManual)
	for _, w := range res.Warnings {
		fmt.Fprintf(c.env.Stderr, "warning: %v\n", w)
	}
	if err != nil {
		return c.failf(exitFailure, "backup failed: %v", err)
	}
	if res.Skipped {
		return c.failf(exitFailure, "backup skipped: %s", res.Reason)
	}
	c.env.Log.Info("manual backup", "game", g.Name, "path", res.Path)
	fmt.Fprintf(c.env.Stdout, "Backed up to %s\n", res.Path)
	return exitOK
}
