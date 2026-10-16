package cli

import (
	"fmt"
	"path/filepath"

	"autosave-games/internal/config"
)

func (c *command) validate() int {
	cfg, err := config.Load(c.cfgPath)
	if err != nil {
		return c.failf(exitFailure, "%v", err)
	}
	fmt.Fprintln(c.env.Stdout, "OK")
	for _, g := range cfg.Games {
		for _, b := range config.ResolveSaves(g.Saves) {
			p := b.AbsBase
			if b.Pattern != "" {
				p = filepath.Join(b.AbsBase, b.Pattern)
			}
			fmt.Fprintf(c.env.Stdout, "%s: %s\n", g.Name, p)
		}
	}
	return exitOK
}
