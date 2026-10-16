// Package tray implements the system tray menu of autosave-tray.exe.
package tray

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

// MutexName guards against running two tray apps in one user session.
const MutexName = `Local\autosave-games`

// AcquireSingleInstance creates the named mutex. already is true when
// another holder exists; release closes this process's handle.
func AcquireSingleInstance(name string) (release func(), already bool, err error) {
	ptr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, fmt.Errorf("mutex name: %w", err)
	}
	h, err := windows.CreateMutex(nil, false, ptr)
	if h == 0 {
		return nil, false, fmt.Errorf("create mutex: %w", err)
	}
	release = func() { windows.CloseHandle(h) }
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return release, true, nil
	}
	return release, false, nil
}
