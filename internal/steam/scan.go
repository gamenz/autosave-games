package steam

import (
	"os"
	"sort"
	"strings"
)

// redistributablesAppID is "Steamworks Common Redistributables", not a game.
const redistributablesAppID = 228980

// Skip reasons.
const (
	ReasonRedistributable = "redistributable"
	ReasonNotInstalled    = "not installed"
	ReasonDuplicate       = "duplicate"
	ReasonNoExecutable    = "no executable"
)

// Found is one installed game ready to be written to games.yaml.
// Saves is empty when the cloud cache has no files for it.
type Found struct {
	AppID      int
	Name       string
	InstallDir string
	Processes  []string
	Saves      []string
}

// Skip is an app that was not turned into a game.
type Skip struct {
	AppID  int
	Name   string
	Reason string
}

// Result is the outcome of Scan.
type Result struct {
	Found    []Found
	Skipped  []Skip
	Warnings []string
}

// Scan discovers installed games in every library of the Steam install at
// steamRoot. Apps sharing an install dir become one game (lowest appid).
func Scan(steamRoot string) (Result, error) {
	var res Result
	libs, err := Libraries(steamRoot)
	if err != nil {
		return res, err
	}
	groups := map[string][]Manifest{}
	var order []string
	seen := map[int]bool{}
	for _, lib := range libs {
		manifests, errs := Manifests(lib)
		for _, e := range errs {
			res.Warnings = append(res.Warnings, e.Error())
		}
		for _, m := range manifests {
			if reason := skipReason(m, seen); reason != "" {
				res.Skipped = append(res.Skipped, Skip{AppID: m.AppID, Name: m.Name, Reason: reason})
				continue
			}
			seen[m.AppID] = true
			key := strings.ToLower(m.InstallDir)
			if groups[key] == nil {
				order = append(order, key)
			}
			groups[key] = append(groups[key], m)
		}
	}
	accounts := Accounts(steamRoot)
	for _, key := range order {
		found, skip, warnings := buildGame(steamRoot, groups[key], accounts)
		res.Warnings = append(res.Warnings, warnings...)
		if skip != nil {
			res.Skipped = append(res.Skipped, *skip)
			continue
		}
		res.Found = append(res.Found, found)
	}
	sort.Slice(res.Found, func(i, j int) bool {
		return strings.ToLower(res.Found[i].Name) < strings.ToLower(res.Found[j].Name)
	})
	return res, nil
}

func skipReason(m Manifest, seen map[int]bool) string {
	if m.AppID == redistributablesAppID {
		return ReasonRedistributable
	}
	if info, err := os.Stat(m.InstallDir); err != nil || !info.IsDir() {
		return ReasonNotInstalled
	}
	if seen[m.AppID] {
		return ReasonDuplicate
	}
	return ""
}

// buildGame merges a group of manifests sharing one install dir.
func buildGame(steamRoot string, group []Manifest, accounts []string) (Found, *Skip, []string) {
	sort.Slice(group, func(i, j int) bool { return group[i].AppID < group[j].AppID })
	primary := group[0]
	exes, err := Executables(primary.InstallDir)
	if err != nil || len(exes) == 0 {
		return Found{}, &Skip{AppID: primary.AppID, Name: primary.Name, Reason: ReasonNoExecutable}, nil
	}
	f := Found{AppID: primary.AppID, Name: primary.Name, InstallDir: primary.InstallDir, Processes: exes}
	var warnings []string
	uniq := map[string]bool{}
	for _, m := range group {
		files, errs := CloudFiles(steamRoot, m.AppID)
		for _, e := range errs {
			warnings = append(warnings, e.Error())
		}
		saves, w := SaveEntries(steamRoot, primary.InstallDir, m.AppID, files, accounts)
		warnings = append(warnings, w...)
		for _, s := range saves {
			if !uniq[strings.ToLower(s)] {
				uniq[strings.ToLower(s)] = true
				f.Saves = append(f.Saves, s)
			}
		}
	}
	return f, nil, warnings
}
