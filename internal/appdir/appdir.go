// Package appdir locates the folder of the running executable, where the
// portable config and log files live next to autosave.exe and autosave-tray.exe.
package appdir

import (
	"os"
	"path/filepath"
)

// Dir returns the directory of the running executable, or "." when it
// cannot be determined.
func Dir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}
