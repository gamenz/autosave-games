package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"autosave-games/configs"
)

// EnsureConfig writes the embedded example config when path does not exist.
func EnsureConfig(path string) (created bool, err error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("check config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, configs.Example, 0o644); err != nil {
		return false, fmt.Errorf("write example config: %w", err)
	}
	return true, nil
}
