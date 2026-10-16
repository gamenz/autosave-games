package steam

import (
	"os"
	"testing"
)

// TestScanRealSteam prints what a scan of this machine's Steam would add.
// Read-only; enable with AUTOSAVE_STEAM_TEST=1.
func TestScanRealSteam(t *testing.T) {
	if os.Getenv("AUTOSAVE_STEAM_TEST") != "1" {
		t.Skip("set AUTOSAVE_STEAM_TEST=1 to scan the real Steam install")
	}
	root, err := LocateRoot()
	if err != nil {
		t.Fatal(err)
	}
	res, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Steam root %s: %d found, %d skipped, %d warnings", root, len(res.Found), len(res.Skipped), len(res.Warnings))
	for _, f := range res.Found {
		t.Logf("FOUND %d %q procs=%v saves=%v", f.AppID, f.Name, f.Processes, f.Saves)
	}
	for _, s := range res.Skipped {
		t.Logf("SKIP  %d %q %s", s.AppID, s.Name, s.Reason)
	}
	for _, w := range res.Warnings {
		t.Logf("WARN  %s", w)
	}
}
