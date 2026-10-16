package app

import (
	"fmt"
	"strings"

	"autosave-games/internal/config"
	"autosave-games/internal/steam"
)

// ScanSteam adds installed Steam games to the config file and reloads it.
func (a *App) ScanSteam() (config.AddSummary, error) {
	locate := a.deps.SteamRoot
	if locate == nil {
		locate = steam.LocateRoot
	}
	log := a.deps.Log
	root, err := locate()
	if err != nil {
		return config.AddSummary{}, err
	}
	res, err := steam.Scan(root)
	if err != nil {
		return config.AddSummary{}, err
	}
	for _, s := range res.Skipped {
		log.Info("steam scan", "appid", s.AppID, "name", s.Name, "result", "skipped", "reason", s.Reason)
	}
	for _, w := range res.Warnings {
		log.Warn("steam scan", "warning", w)
	}
	games := make([]config.NewGame, len(res.Found))
	for i, f := range res.Found {
		games[i] = config.NewGame{Name: f.Name, SteamAppID: f.AppID, InstallDir: f.InstallDir,
			Processes: f.Processes, Saves: f.Saves}
	}
	sum, err := config.AddGames(a.deps.ConfigPath, games)
	if err != nil {
		return sum, err
	}
	logResults(a, res.Found, sum)
	a.ReloadConfig()
	log.Info("steam scan done", "root", root, "summary", FormatScanSummary(sum))
	return sum, nil
}

func logResults(a *App, found []steam.Found, sum config.AddSummary) {
	result := map[string]string{}
	for _, n := range sum.Added {
		result[n] = "added"
	}
	for _, n := range sum.NeedSaves {
		result[n] = "needs-saves"
	}
	for _, n := range sum.Configured {
		result[n] = "configured"
	}
	for _, f := range found {
		a.deps.Log.Info("steam scan", "appid", f.AppID, "name", f.Name, "result", result[f.Name],
			"saves", strings.Join(f.Saves, "; "))
	}
}

// FormatScanSummary renders the toast body for a scan.
func FormatScanSummary(s config.AddSummary) string {
	return fmt.Sprintf("added %d games, %d need save paths, %d already configured",
		len(s.Added), len(s.NeedSaves), len(s.Configured))
}
