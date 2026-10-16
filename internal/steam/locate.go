package steam

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// ErrNotFound means no Steam installation was found.
var ErrNotFound = errors.New("Steam not found")

// LocateRoot returns the Steam installation folder from the registry.
func LocateRoot() (string, error) {
	candidates := []struct {
		root  registry.Key
		path  string
		value string
	}{
		{registry.CURRENT_USER, `Software\Valve\Steam`, "SteamPath"},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Valve\Steam`, "InstallPath"},
	}
	for _, c := range candidates {
		k, err := registry.OpenKey(c.root, c.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		v, _, err := k.GetStringValue(c.value)
		k.Close()
		if err != nil || v == "" {
			continue
		}
		dir := filepath.Clean(filepath.FromSlash(v))
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, nil
		}
	}
	return "", ErrNotFound
}
