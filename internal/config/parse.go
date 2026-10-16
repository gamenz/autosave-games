package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse decodes YAML, applies defaults, expands paths and fills slugs.
// It does not validate; call Validate on the result.
func Parse(data []byte, env EnvFunc) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyDefaults(&c)
	if c.BackupRoot != "" {
		c.BackupRoot = ExpandPath(c.BackupRoot, env)
	}
	for i := range c.Games {
		g := &c.Games[i]
		g.Slug = Slugify(g.Name)
		for j, p := range g.Processes {
			g.Processes[j] = strings.ToLower(strings.TrimSpace(p))
		}
		for j, s := range g.Saves {
			g.Saves[j] = ExpandPath(s, env)
		}
	}
	return &c, nil
}

// Load reads, parses and validates the config file at path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	c, err := Parse(data, os.Getenv)
	if err != nil {
		return nil, err
	}
	if err := Validate(c); err != nil {
		return nil, err
	}
	return c, nil
}

// DefaultPath is %APPDATA%\autosave-games\games.yaml.
func DefaultPath() string {
	return filepath.Join(os.Getenv("APPDATA"), "autosave-games", "games.yaml")
}

func applyDefaults(c *Config) {
	if c.PollSeconds == 0 {
		c.PollSeconds = DefaultPollSeconds
	}
	if c.Notify == "" {
		c.Notify = DefaultNotify
	}
	if c.Defaults.KeepLast == 0 {
		c.Defaults.KeepLast = DefaultKeepLast
	}
	if c.Defaults.SettleSeconds == 0 {
		c.Defaults.SettleSeconds = DefaultSettleSeconds
	}
	for i := range c.Games {
		g := &c.Games[i]
		if g.KeepLast == 0 {
			g.KeepLast = c.Defaults.KeepLast
		}
		if g.SettleSeconds == 0 {
			g.SettleSeconds = c.Defaults.SettleSeconds
		}
	}
}
