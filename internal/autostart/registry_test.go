package autostart

import (
	"os"
	"testing"
)

func TestAutostartRoundTrip(t *testing.T) {
	if os.Getenv("AUTOSAVE_REGISTRY_TEST") != "1" {
		t.Skip("set AUTOSAVE_REGISTRY_TEST=1 to touch HKCU Run key")
	}
	e := Entry{ValueName: "autosave-games-test"}
	defer e.Disable()
	if err := e.Enable(`C:\x\autosave-tray.exe`); err != nil {
		t.Fatal(err)
	}
	if on, err := e.Enabled(); err != nil || !on {
		t.Fatalf("Enabled = %v, %v", on, err)
	}
	if err := e.Disable(); err != nil {
		t.Fatal(err)
	}
	if on, _ := e.Enabled(); on {
		t.Error("still enabled after Disable")
	}
	if err := e.Disable(); err != nil {
		t.Errorf("second Disable: %v", err)
	}
}
