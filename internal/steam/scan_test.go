package steam

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"
)

func cloudVDF(appid int, files map[string]int) string {
	s := fmt.Sprintf("\"%d\"\n{\n", appid)
	for p, root := range files {
		s += fmt.Sprintf("\t\"%s\"\n\t{\n\t\t\"root\"\t\"%d\"\n\t}\n", p, root)
	}
	return s + "}\n"
}

// buildSteamTree creates a fake Steam install with two libraries.
func buildSteamTree(t *testing.T) string {
	t.Helper()
	root, lib2 := t.TempDir(), t.TempDir()
	apps := filepath.Join(root, "steamapps")
	common := filepath.Join(apps, "common")
	writeFile(t, filepath.Join(apps, "libraryfolders.vdf"), fmt.Sprintf(
		`"libraryfolders" { "0" { "path" "%s" } "1" { "path" "%s" } }`, vdfPath(root), vdfPath(lib2)))

	writeFile(t, filepath.Join(apps, "appmanifest_1245620.acf"), acf(1245620, "ELDEN RING", "ELDEN RING"))
	writeFile(t, filepath.Join(common, "ELDEN RING", "Game", "eldenring.exe"), "")
	writeFile(t, filepath.Join(root, "userdata", "111", "1245620", "remotecache.vdf"),
		cloudVDF(1245620, map[string]int{"EldenRing/76561199478888042/ER0000.sl2": 4}))
	writeFile(t, filepath.Join(root, "userdata", "222", "1245620", "remotecache.vdf"),
		cloudVDF(1245620, map[string]int{"EldenRing/76561198000000001/ER0000.sl2": 4}))

	writeFile(t, filepath.Join(apps, "appmanifest_228980.acf"), acf(228980, "Steamworks Common Redistributables", "Steamworks Shared"))
	writeFile(t, filepath.Join(common, "Steamworks Shared", "x.exe"), "")

	writeFile(t, filepath.Join(apps, "appmanifest_220.acf"), acf(220, "Half-Life 2", "Half-Life 2"))
	writeFile(t, filepath.Join(apps, "appmanifest_380.acf"), acf(380, "Half-Life 2: Episode One", "Half-Life 2"))
	writeFile(t, filepath.Join(common, "Half-Life 2", "hl2.exe"), "")
	writeFile(t, filepath.Join(root, "userdata", "111", "220", "remotecache.vdf"),
		cloudVDF(220, map[string]int{"hl2/save/a.sav": 1}))
	writeFile(t, filepath.Join(root, "userdata", "111", "380", "remotecache.vdf"),
		cloudVDF(380, map[string]int{"s.sav": 0}))

	writeFile(t, filepath.Join(apps, "appmanifest_999.acf"), acf(999, "Soundtrack", "Soundtrack"))
	writeFile(t, filepath.Join(common, "Soundtrack", "track.mp3"), "")

	apps2 := filepath.Join(lib2, "steamapps")
	writeFile(t, filepath.Join(apps2, "appmanifest_255710.acf"), acf(255710, "Cities: Skylines", "Cities_Skylines"))
	writeFile(t, filepath.Join(apps2, "common", "Cities_Skylines", "Cities.exe"), "")
	writeFile(t, filepath.Join(apps2, "appmanifest_1245620.acf"), acf(1245620, "ELDEN RING", "ELDEN RING"))
	return root
}

func TestScan(t *testing.T) {
	root := buildSteamTree(t)
	res, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	byName := map[string]Found{}
	for _, f := range res.Found {
		names = append(names, f.Name)
		byName[f.Name] = f
	}
	if want := []string{"Cities: Skylines", "ELDEN RING", "Half-Life 2"}; !slices.Equal(names, want) {
		t.Fatalf("found = %v, want %v (skipped %+v)", names, want, res.Skipped)
	}
	er := byName["ELDEN RING"]
	if er.AppID != 1245620 || !slices.Equal(er.Processes, []string{"eldenring.exe"}) ||
		!slices.Equal(er.Saves, []string{"%APPDATA%/EldenRing"}) {
		t.Errorf("ELDEN RING = %+v", er)
	}
	if c := byName["Cities: Skylines"]; len(c.Saves) != 0 || c.Processes[0] != "cities.exe" {
		t.Errorf("Cities = %+v", c)
	}
	hl2 := byName["Half-Life 2"]
	wantHL2 := []string{
		filepath.Join(root, "steamapps", "common", "Half-Life 2") + "/hl2/save",
		filepath.Join(root, "userdata", "*", "380", "remote"),
	}
	if hl2.AppID != 220 || !slices.Equal(hl2.Saves, wantHL2) {
		t.Errorf("Half-Life 2 = %+v, want saves %v", hl2, wantHL2)
	}
	reasons := map[int]string{}
	for _, s := range res.Skipped {
		reasons[s.AppID] = s.Reason
	}
	want := map[int]string{228980: "redistributable", 999: "no executable", 1245620: "not installed"}
	for id, r := range want {
		if reasons[id] != r {
			t.Errorf("skip %d = %q, want %q (all %+v)", id, reasons[id], r, res.Skipped)
		}
	}
}

func TestScanMissingRoot(t *testing.T) {
	res, err := Scan(filepath.Join(t.TempDir(), "nope"))
	if err != nil || len(res.Found) != 0 {
		t.Fatalf("Scan = %+v, %v", res, err)
	}
}

func TestScanDropsSharedExeNames(t *testing.T) {
	root := t.TempDir()
	common := filepath.Join(root, "steamapps", "common")
	for _, g := range []struct {
		id        int
		name, dir string
		exe       string
	}{{1, "One", "One", "one.exe"}, {2, "Two", "Two", "two.exe"}} {
		writeFile(t, filepath.Join(root, "steamapps", fmt.Sprintf("appmanifest_%d.acf", g.id)), acf(g.id, g.name, g.dir))
		writeFile(t, filepath.Join(common, g.dir, g.exe), "")
		writeFile(t, filepath.Join(common, g.dir, "helper.exe"), "")
		writeFile(t, filepath.Join(common, g.dir, "InstallerMessage.exe"), "")
	}
	res, err := Scan(root)
	if err != nil || len(res.Found) != 2 {
		t.Fatalf("Scan = %+v, %v", res, err)
	}
	if !slices.Equal(res.Found[0].Processes, []string{"one.exe"}) || !slices.Equal(res.Found[1].Processes, []string{"two.exe"}) {
		t.Errorf("processes = %v / %v, want shared helper.exe and installer dropped", res.Found[0].Processes, res.Found[1].Processes)
	}
}
