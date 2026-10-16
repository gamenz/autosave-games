// Package cli implements the autosave.exe subcommands.
package cli

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"autosave-games/internal/app"
	"autosave-games/internal/backup"
	"autosave-games/internal/config"
	"autosave-games/internal/process"
)

// Exit codes.
const (
	exitOK      = 0
	exitFailure = 1
	exitUsage   = 2
)

const usage = `usage:
  autosave [--config path] list [game]
  autosave [--config path] backup <game>
  autosave [--config path] restore <game> [--at <prefix>] [--yes]
  autosave [--config path] validate
  autosave autostart on|off
`

// Env carries the process environment so commands are testable.
type Env struct {
	Stdout, Stderr io.Writer
	Stdin          io.Reader
	Lister         process.Lister
	Log            *slog.Logger
	Now            func() time.Time
	Version        string
}

type command struct {
	env     Env
	cfgPath string
}

// Run executes one CLI invocation and returns the process exit code.
func Run(args []string, e Env) int {
	fs := flag.NewFlagSet("autosave", flag.ContinueOnError)
	fs.SetOutput(e.Stderr)
	cfgPath := fs.String("config", config.DefaultPath(), "path to games.yaml")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprint(e.Stderr, usage)
		return exitUsage
	}
	c := &command{env: e, cfgPath: *cfgPath}
	switch rest[0] {
	case "list":
		return c.list(rest[1:])
	case "backup":
		return c.backup(rest[1:])
	case "restore":
		return c.restore(rest[1:])
	case "validate":
		return c.validate()
	case "autostart":
		return c.autostart(rest[1:])
	}
	fmt.Fprintf(e.Stderr, "unknown command %q\n%s", rest[0], usage)
	return exitUsage
}

func (c *command) failf(code int, format string, args ...any) int {
	fmt.Fprintf(c.env.Stderr, format+"\n", args...)
	return code
}

// loadConfig creates the example config on first use, then loads it.
func (c *command) loadConfig() (*config.Config, error) {
	created, err := app.EnsureConfig(c.cfgPath)
	if err != nil {
		return nil, err
	}
	if created {
		fmt.Fprintf(c.env.Stdout, "Created example config at %s\n", c.cfgPath)
	}
	return config.Load(c.cfgPath)
}

// loadGame loads the config and resolves query to a game.
func (c *command) loadGame(query string) (*config.Config, *config.Game, int) {
	cfg, err := c.loadConfig()
	if err != nil {
		return nil, nil, c.failf(exitFailure, "%v", err)
	}
	g, ok := cfg.FindGame(query)
	if !ok {
		slugs := make([]string, len(cfg.Games))
		for i, x := range cfg.Games {
			slugs[i] = x.Slug
		}
		return nil, nil, c.failf(exitUsage, "unknown game %q; known: %s", query, strings.Join(slugs, ", "))
	}
	return cfg, g, exitOK
}

func (c *command) service(cfg *config.Config) *backup.Service {
	return &backup.Service{Root: cfg.BackupRoot, AppVersion: c.env.Version, Now: c.env.Now}
}

// parseInterspersed parses flags that may appear after positional args.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}
