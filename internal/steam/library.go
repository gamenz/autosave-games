package steam

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Manifest is one installed app from steamapps\appmanifest_<id>.acf.
type Manifest struct {
	AppID      int
	Name       string
	InstallDir string // absolute: <library>\steamapps\common\<installdir>
}

// Libraries returns root plus every library listed in libraryfolders.vdf,
// cleaned and de-duplicated (case-insensitive). A missing file yields [root].
func Libraries(root string) ([]string, error) {
	libs := []string{filepath.Clean(root)}
	n, err := ReadVDF(filepath.Join(root, "steamapps", "libraryfolders.vdf"))
	if errors.Is(err, fs.ErrNotExist) {
		return libs, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Steam libraries: %w", err)
	}
	seen := map[string]bool{strings.ToLower(libs[0]): true}
	folders := n.Child("libraryfolders")
	if folders == nil {
		return libs, nil
	}
	for _, entry := range folders.Children {
		p := entry.Get("path")
		if p == "" {
			continue
		}
		p = filepath.Clean(p)
		if key := strings.ToLower(p); !seen[key] {
			seen[key] = true
			libs = append(libs, p)
		}
	}
	return libs, nil
}

// Manifests reads every app manifest of library, sorted by AppID. Broken
// files are reported and skipped.
func Manifests(library string) ([]Manifest, []error) {
	files, err := filepath.Glob(filepath.Join(library, "steamapps", "appmanifest_*.acf"))
	if err != nil {
		return nil, []error{err}
	}
	var out []Manifest
	var errs []error
	for _, f := range files {
		m, err := readManifest(library, f)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AppID < out[j].AppID })
	return out, errs
}

func readManifest(library, path string) (Manifest, error) {
	n, err := ReadVDF(path)
	if err != nil {
		return Manifest{}, err
	}
	state := n.Child("AppState")
	id, err := strconv.Atoi(state.Get("appid"))
	dir := state.Get("installdir")
	if err != nil || dir == "" {
		return Manifest{}, fmt.Errorf("%s: missing appid or installdir", path)
	}
	return Manifest{
		AppID:      id,
		Name:       state.Get("name"),
		InstallDir: filepath.Join(library, "steamapps", "common", dir),
	}, nil
}
