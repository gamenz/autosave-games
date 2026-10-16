package steam

import (
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxExeDepth = 4  // a top-level file has depth 1
	maxExeNames = 10 // names written per game
)

// exeExcludes drops installers, redistributables and crash reporters.
var exeExcludes = []string{
	"*setup*", "*redist*", "*prereq*", "vc_redist*", "dxsetup*", "dotnet*", "unins*",
	"crashpad_handler*", "unitycrashhandler*", "*crashreport*", "*crashhandler*", "*bugreport*",
}

type exeFile struct {
	name  string
	depth int
}

// Executables returns up to 10 lower-cased exe names found in installDir
// (depth ≤ 4), shallowest first, excluding helper programs.
func Executables(installDir string) ([]string, error) {
	var found []exeFile
	seen := map[string]bool{}
	err := filepath.WalkDir(installDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(installDir, p)
		depth := len(strings.Split(rel, string(filepath.Separator)))
		if d.IsDir() {
			if rel != "." && depth >= maxExeDepth {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		if depth > maxExeDepth || !strings.HasSuffix(name, ".exe") || excludedExe(name) || seen[name] {
			return nil
		}
		seen[name] = true
		found = append(found, exeFile{name: name, depth: depth})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].depth != found[j].depth {
			return found[i].depth < found[j].depth
		}
		return found[i].name < found[j].name
	})
	var names []string
	for i := 0; i < len(found) && i < maxExeNames; i++ {
		names = append(names, found[i].name)
	}
	return names, nil
}

func excludedExe(name string) bool {
	for _, pattern := range exeExcludes {
		if ok, _ := path.Match(pattern, name); ok {
			return true
		}
	}
	return false
}
