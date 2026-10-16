// Package autostart registers the tray app under the HKCU Run key.
package autostart

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// RunKey is the per-user autostart key (no admin rights needed).
const RunKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// DefaultValueName is the registry value used by the app.
const DefaultValueName = "autosave-games"

// Entry is one value under RunKey.
type Entry struct {
	ValueName string
}

// Default returns the app's autostart entry.
func Default() Entry { return Entry{ValueName: DefaultValueName} }

// Enable starts exePath at user logon.
func (e Entry) Enable(exePath string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, RunKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer k.Close()
	if err := k.SetStringValue(e.ValueName, `"`+exePath+`"`); err != nil {
		return fmt.Errorf("set autostart: %w", err)
	}
	return nil
}

// Disable removes the entry; a missing entry is not an error.
func (e Entry) Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, RunKey, registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer k.Close()
	if err := k.DeleteValue(e.ValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("remove autostart: %w", err)
	}
	return nil
}

// Enabled reports whether the entry exists.
func (e Entry) Enabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, RunKey, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open Run key: %w", err)
	}
	defer k.Close()
	if _, _, err := k.GetStringValue(e.ValueName); err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read autostart: %w", err)
	}
	return true, nil
}
