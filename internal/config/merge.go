package config

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// NewGame is a discovered game to add to games.yaml (paths unexpanded).
type NewGame struct {
	Name       string
	SteamAppID int
	InstallDir string
	Processes  []string
	Saves      []string // empty: added as a commented block to fill in
}

// AddSummary lists game names by outcome.
type AddSummary struct {
	Added      []string
	NeedSaves  []string
	Configured []string
}

// AddGames appends games that are not configured yet to the config at path,
// keeping existing content and comments. Games without saves are appended as
// commented blocks. Nothing is written when there is nothing to add.
func AddGames(path string, games []NewGame) (AddSummary, error) {
	var sum AddSummary
	current, err := Load(path)
	if err != nil {
		return sum, fmt.Errorf("config is invalid, fix it first: %w", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return sum, fmt.Errorf("read config: %w", err)
	}
	text := normalizeText(raw)

	var added, needSaves []NewGame
	for _, g := range games {
		switch {
		case isConfigured(current, text, g):
			sum.Configured = append(sum.Configured, g.Name)
		case len(g.Saves) > 0:
			added = append(added, g)
			sum.Added = append(sum.Added, g.Name)
		default:
			needSaves = append(needSaves, g)
			sum.NeedSaves = append(sum.NeedSaves, g.Name)
		}
	}
	if len(added)+len(needSaves) == 0 {
		return sum, nil
	}
	out, err := mergeYAML(text, added, needSaves)
	if err != nil {
		return sum, err
	}
	if err := writeWithBackup(path, raw, out); err != nil {
		return sum, err
	}
	return sum, nil
}

func normalizeText(raw []byte) string {
	raw = bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

// isConfigured reports whether g matches an existing game (appid, install
// dir or slug) or a commented block from an earlier scan.
func isConfigured(c *Config, text string, g NewGame) bool {
	dir := ""
	if strings.TrimSpace(g.InstallDir) != "" {
		dir = ExpandPath(g.InstallDir, os.Getenv)
	}
	slug := Slugify(g.Name)
	for _, e := range c.Games {
		if g.SteamAppID != 0 && e.SteamAppID == g.SteamAppID {
			return true
		}
		if dir != "" && e.InstallDir != "" && IsWithin(dir, e.InstallDir) && IsWithin(e.InstallDir, dir) {
			return true
		}
		if slug != "" && e.Slug == slug {
			return true
		}
	}
	if g.SteamAppID == 0 {
		return false
	}
	marker := regexp.MustCompile(`steam_appid[ :]+` + strconv.Itoa(g.SteamAppID) + `\b`)
	return marker.MatchString(text)
}

// writeWithBackup saves the original bytes to path.bak, then replaces path
// atomically with content.
func writeWithBackup(path string, original []byte, content string) error {
	if err := os.WriteFile(path+".bak", original, 0o644); err != nil {
		return fmt.Errorf("backup config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}
