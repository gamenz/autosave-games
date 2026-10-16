package session

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestCanOpenExclusiveLock(t *testing.T) {
	p := writeFile(t, t.TempDir(), "locked.sav", "x")
	h, err := windows.CreateFile(windows.StringToUTF16Ptr(p),
		windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	if CanOpen(p) {
		t.Error("exclusively locked file should not open")
	}
	windows.CloseHandle(h)
	if !CanOpen(p) {
		t.Error("file should open after unlock")
	}
}
