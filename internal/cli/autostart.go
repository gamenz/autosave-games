package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"autosave-games/internal/autostart"
)

// trayExeName is the tray binary expected next to autosave.exe.
const trayExeName = "autosave-tray.exe"

func (c *command) autostart(args []string) int {
	if len(args) != 1 || (args[0] != "on" && args[0] != "off") {
		return c.failf(exitUsage, "usage: autosave autostart on|off")
	}
	entry := autostart.Default()
	if args[0] == "off" {
		if err := entry.Disable(); err != nil {
			return c.failf(exitFailure, "%v", err)
		}
		fmt.Fprintln(c.env.Stdout, "Autostart disabled")
		return exitOK
	}
	exe, err := os.Executable()
	if err != nil {
		return c.failf(exitFailure, "locate executable: %v", err)
	}
	tray := filepath.Join(filepath.Dir(exe), trayExeName)
	if _, err := os.Stat(tray); err != nil {
		return c.failf(exitFailure, "%s not found next to %s", trayExeName, filepath.Base(exe))
	}
	if err := entry.Enable(tray); err != nil {
		return c.failf(exitFailure, "%v", err)
	}
	fmt.Fprintf(c.env.Stdout, "Autostart enabled: %s\n", tray)
	return exitOK
}
