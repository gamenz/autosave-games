package steam

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLibraries(t *testing.T) {
	root := t.TempDir()
	other := filepath.Join(t.TempDir(), "Trò chơi", "SteamLibrary")
	writeFile(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"), fmt.Sprintf(
		`"libraryfolders" { "0" { "path" "%s" } "1" { "path" "%s" } }`,
		vdfPath(strings.ToUpper(root)), vdfPath(other)))
	libs, err := Libraries(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{root, other}; !slices.Equal(libs, want) {
		t.Fatalf("Libraries = %v, want %v", libs, want)
	}
}

func TestLibrariesWithoutFile(t *testing.T) {
	root := t.TempDir()
	libs, err := Libraries(root)
	if err != nil || !slices.Equal(libs, []string{root}) {
		t.Fatalf("Libraries = %v, %v", libs, err)
	}
}

func TestManifests(t *testing.T) {
	lib := t.TempDir()
	writeFile(t, filepath.Join(lib, "steamapps", "appmanifest_1245620.acf"), acf(1245620, "ELDEN RING", "ELDEN RING"))
	writeFile(t, filepath.Join(lib, "steamapps", "appmanifest_220.acf"), acf(220, "Half-Life 2", "Half-Life 2"))
	writeFile(t, filepath.Join(lib, "steamapps", "appmanifest_9.acf"), `"AppState" {`)
	got, errs := Manifests(lib)
	if len(errs) != 1 {
		t.Errorf("errs = %v, want 1", errs)
	}
	want := []Manifest{
		{AppID: 220, Name: "Half-Life 2", InstallDir: filepath.Join(lib, "steamapps", "common", "Half-Life 2")},
		{AppID: 1245620, Name: "ELDEN RING", InstallDir: filepath.Join(lib, "steamapps", "common", "ELDEN RING")},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Manifests = %+v, want %+v", got, want)
	}
}

func TestLibrariesWithoutBlock(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"), `"other" { }`)
	libs, err := Libraries(root)
	if err != nil || !slices.Equal(libs, []string{root}) {
		t.Fatalf("Libraries = %v, %v", libs, err)
	}
}
