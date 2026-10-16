package cli

import (
	"bufio"
	"flag"
	"fmt"
	"strings"

	"autosave-games/internal/backup"
	"autosave-games/internal/process"
)

func (c *command) restore(args []string) int {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(c.env.Stderr)
	at := fs.String("at", "", "archive name prefix (default: latest)")
	yes := fs.Bool("yes", false, "do not ask for confirmation")
	positional, err := parseInterspersed(fs, args)
	if err != nil || len(positional) != 1 {
		return c.failf(exitUsage, "usage: autosave restore <game> [--at <prefix>] [--yes]")
	}
	cfg, g, code := c.loadGame(positional[0])
	if code != exitOK {
		return code
	}
	names, err := c.env.Lister.Names()
	if err != nil {
		return c.failf(exitFailure, "cannot check running processes: %v", err)
	}
	if process.AnyRunning(names, g.Processes) {
		return c.failf(exitFailure, "%q is running; close it first", g.Name)
	}
	svc := c.service(cfg)
	archive, err := backup.FindArchive(svc.GameDir(*g), *at)
	if err != nil {
		return c.failf(exitFailure, "%v", err)
	}
	targets, err := backup.PlanRestore(archive.Path)
	if err != nil {
		return c.failf(exitFailure, "%v", err)
	}
	fmt.Fprintf(c.env.Stdout, "Restore %s to:\n", archive.Name)
	for _, t := range targets {
		fmt.Fprintf(c.env.Stdout, "  %s\n", t.AbsPath)
	}
	if !*yes && !c.confirm() {
		fmt.Fprintln(c.env.Stdout, "aborted")
		return exitFailure
	}
	if _, err := svc.Run(*g, backup.KindPreRestore); err != nil {
		return c.failf(exitFailure, "prerestore backup failed, nothing restored: %v", err)
	}
	written, err := backup.Restore(archive.Path)
	if err != nil {
		return c.failf(exitFailure, "restore failed after %d files: %v", len(written), err)
	}
	c.env.Log.Info("restored", "game", g.Name, "archive", archive.Name, "files", len(written))
	fmt.Fprintf(c.env.Stdout, "Restored %d files\n", len(written))
	return exitOK
}

func (c *command) confirm() bool {
	fmt.Fprint(c.env.Stdout, "Continue? [y/N] ")
	line, _ := bufio.NewReader(c.env.Stdin).ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
