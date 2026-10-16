// Package config loads, expands and validates the games.yaml configuration.
package config

import "strings"

// Notify controls which events produce a desktop notification.
type Notify string

const (
	NotifyAll    Notify = "all"
	NotifyErrors Notify = "errors"
	NotifyNone   Notify = "none"
)

// Default values applied when a field is omitted (zero).
const (
	DefaultPollSeconds   = 3
	DefaultKeepLast      = 20
	DefaultSettleSeconds = 5
	DefaultNotify        = NotifyErrors
)

// EnvFunc resolves an environment variable; os.Getenv in production.
type EnvFunc func(string) string

// Config is the root of games.yaml.
type Config struct {
	BackupRoot  string   `yaml:"backup_root"`
	PollSeconds int      `yaml:"poll_seconds"`
	Notify      Notify   `yaml:"notify"`
	Defaults    Defaults `yaml:"defaults"`
	Games       []Game   `yaml:"games"`
}

// Defaults holds per-game values used when a game does not override them.
type Defaults struct {
	KeepLast      int `yaml:"keep_last"`
	SettleSeconds int `yaml:"settle_seconds"`
}

// Game describes one tracked game. Saves hold expanded paths after Parse.
type Game struct {
	Name          string   `yaml:"name"`
	Slug          string   `yaml:"-"`
	Processes     []string `yaml:"processes"`
	Saves         []string `yaml:"saves"`
	Exclude       []string `yaml:"exclude"`
	SteamAppID    int      `yaml:"steam_appid"`
	InstallDir    string   `yaml:"install_dir"` // expanded; any exe inside counts as running
	KeepLast      int      `yaml:"keep_last"`
	SettleSeconds int      `yaml:"settle_seconds"`
}

// FindGame returns the game whose name or slug matches query, ignoring case.
func (c *Config) FindGame(query string) (*Game, bool) {
	q := strings.TrimSpace(query)
	for i := range c.Games {
		g := &c.Games[i]
		if strings.EqualFold(g.Name, q) || strings.EqualFold(g.Slug, q) {
			return g, true
		}
	}
	return nil, false
}
