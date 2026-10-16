package steam

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// CloudFile is one file listed in a Steam Cloud remotecache.vdf.
type CloudFile struct {
	Account string
	Root    int
	Path    string // forward slashes, no leading "./" or "/"
}

// rootBases maps remotecache "root" codes to the folder written in games.yaml.
// Root 0 (Steam's own remote folder) is handled separately.
var rootBases = map[int]string{
	2: "%USERPROFILE%/Documents",
	3: "%LOCALAPPDATA%",
	4: "%APPDATA%",
}

var steamIDPattern = regexp.MustCompile(`^7656119\d{10}$`)

// Accounts lists the account folders under <steamRoot>\userdata, sorted.
func Accounts(steamRoot string) []string {
	entries, err := os.ReadDir(filepath.Join(steamRoot, "userdata"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// CloudFiles reads the cloud cache of appID from every account.
func CloudFiles(steamRoot string, appID int) ([]CloudFile, []error) {
	var files []CloudFile
	var errs []error
	for _, acct := range Accounts(steamRoot) {
		p := filepath.Join(steamRoot, "userdata", acct, strconv.Itoa(appID), "remotecache.vdf")
		if _, err := os.Stat(p); err != nil {
			continue
		}
		n, err := ReadVDF(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		top := n.Child(strconv.Itoa(appID))
		if top == nil {
			continue
		}
		for _, c := range top.Children {
			if !c.Block || c.Get("root") == "" {
				continue
			}
			root, err := strconv.Atoi(c.Get("root"))
			if err != nil {
				continue
			}
			files = append(files, CloudFile{Account: acct, Root: root, Path: normalizePath(c.Key)})
		}
	}
	return files, errs
}

// SaveEntries turns cloud files into "saves" entries (spec §4).
func SaveEntries(steamRoot, installDir string, appID int, files []CloudFile, accounts []string) (saves []string, warnings []string) {
	byRoot := map[int][]string{}
	for _, f := range files {
		byRoot[f.Root] = append(byRoot[f.Root], normalizePath(f.Path))
	}
	roots := make([]int, 0, len(byRoot))
	for r := range byRoot {
		roots = append(roots, r)
	}
	sort.Ints(roots)
	for _, r := range roots {
		switch base, known := rootBases[r]; {
		case r == 0:
			saves = append(saves, filepath.Join(steamRoot, "userdata", "*", strconv.Itoa(appID), "remote"))
		case r == 1:
			saves = append(saves, folderEntries(installDir, byRoot[r], accounts)...)
		case known:
			saves = append(saves, folderEntries(base, byRoot[r], accounts)...)
		default:
			warnings = append(warnings, fmt.Sprintf("appid %d: unknown cloud root %d", appID, r))
		}
	}
	return saves, warnings
}

// folderEntries returns base/<common folder>, or one entry per file when the
// files share no folder below base (never back up a whole root).
func folderEntries(base string, paths []string, accounts []string) []string {
	var common []string
	for i, p := range paths {
		dir := strings.Split(p, "/")
		dir = dir[:len(dir)-1]
		if i == 0 {
			common = dir
			continue
		}
		common = commonPrefix(common, dir)
	}
	for i, seg := range common {
		if isAccountSegment(seg, accounts) {
			common = common[:i]
			break
		}
	}
	if len(common) == 0 {
		uniq := map[string]bool{}
		var out []string
		for _, p := range paths {
			if !uniq[p] {
				uniq[p] = true
				out = append(out, base+"/"+p)
			}
		}
		sort.Strings(out)
		return out
	}
	return []string{base + "/" + strings.Join(common, "/")}
}

func commonPrefix(a, b []string) []string {
	n := 0
	for n < len(a) && n < len(b) && strings.EqualFold(a[n], b[n]) {
		n++
	}
	return a[:n]
}

func isAccountSegment(seg string, accounts []string) bool {
	if steamIDPattern.MatchString(seg) {
		return true
	}
	for _, a := range accounts {
		if seg == a {
			return true
		}
	}
	return false
}

func normalizePath(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	for strings.HasPrefix(p, "./") || strings.HasPrefix(p, "/") {
		p = strings.TrimPrefix(strings.TrimPrefix(p, "./"), "/")
	}
	return p
}
