package config

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const userYAML = "# my config\r\nbackup_root: 'D:/B'\r\ngames: []\r\n# keep me\r\nnotify: all\r\n"

var erGame = NewGame{Name: "ELDEN RING", SteamAppID: 1245620, InstallDir: `D:\Steam\steamapps\common\ELDEN RING`,
	Processes: []string{"eldenring.exe"}, Saves: []string{"%APPDATA%/EldenRing"}}

var citiesGame = NewGame{Name: "Cities: Skylines", SteamAppID: 255710,
	InstallDir: `D:\Steam\steamapps\common\Cities_Skylines`, Processes: []string{"cities.exe"}}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "games.yaml")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func readText(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAddGamesAppendsAndKeepsComments(t *testing.T) {
	original := "\ufeff" + userYAML
	p := writeConfig(t, original)
	sum, err := AddGames(p, []NewGame{erGame, citiesGame})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sum.Added, []string{"ELDEN RING"}) || !slices.Equal(sum.NeedSaves, []string{"Cities: Skylines"}) {
		t.Fatalf("summary = %+v", sum)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatalf("Load after merge: %v\n%s", err, readText(t, p))
	}
	if c.Notify != NotifyAll || len(c.Games) != 1 {
		t.Fatalf("config = %+v", c)
	}
	g := c.Games[0]
	if g.SteamAppID != 1245620 || g.InstallDir != erGame.InstallDir || len(g.Saves) != 1 ||
		!strings.HasSuffix(g.Saves[0], `\EldenRing`) {
		t.Errorf("game = %+v", g)
	}
	text := readText(t, p)
	for _, want := range []string{"# my config", "# keep me", "steam_appid 255710: save path unknown"} {
		if !strings.Contains(text, want) {
			t.Errorf("merged file missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "\r") {
		t.Error("merged file still has CR")
	}
	if bak, _ := os.ReadFile(p + ".bak"); !bytes.Equal(bak, []byte(original)) {
		t.Error(".bak differs from the original")
	}
}

// uncomment enables every commented game block line ("#   ...").
func uncomment(text string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "#   ") {
			lines[i] = strings.TrimPrefix(l, "# ")
		}
	}
	return strings.ReplaceAll(strings.Join(lines, "\n"), "saves: []", "saves: ['x']")
}

func TestCommentedBlockUncommentsToValidYAML(t *testing.T) {
	p := writeConfig(t, "backup_root: 'D:/B'\ngames: []\n")
	tom := NewGame{Name: "Tom's #1 Game", SteamAppID: 42, InstallDir: `D:\G\Tom`, Processes: []string{"tom.exe"}}
	if _, err := AddGames(p, []NewGame{citiesGame, tom}); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err != nil {
		t.Fatalf("merged file invalid: %v", err)
	}
	enabled := writeConfig(t, uncomment(readText(t, p)))
	c, err := Load(enabled)
	if err != nil {
		t.Fatalf("uncommented file invalid: %v\n%s", err, readText(t, enabled))
	}
	var names []string
	for _, g := range c.Games {
		names = append(names, g.Name)
	}
	if !slices.Equal(names, []string{"Cities: Skylines", "Tom's #1 Game"}) {
		t.Errorf("names = %v", names)
	}
}

func TestAddGamesIdempotent(t *testing.T) {
	p := writeConfig(t, userYAML)
	if _, err := AddGames(p, []NewGame{erGame, citiesGame}); err != nil {
		t.Fatal(err)
	}
	before := readText(t, p)
	sum, err := AddGames(p, []NewGame{erGame, citiesGame})
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Added)+len(sum.NeedSaves) != 0 || len(sum.Configured) != 2 {
		t.Errorf("second summary = %+v", sum)
	}
	if readText(t, p) != before {
		t.Error("second run changed the file")
	}
}

func TestAddGamesDedup(t *testing.T) {
	byDir := writeConfig(t, "backup_root: 'D:/B'\ngames:\n  - {name: Mine, install_dir: '"+
		strings.ToUpper(erGame.InstallDir)+"', saves: [x]}\n")
	bySlug := writeConfig(t, "backup_root: 'D:/B'\ngames:\n  - {name: Elden Ring, processes: [a.exe], saves: [x]}\n")
	for _, p := range []string{byDir, bySlug} {
		sum, err := AddGames(p, []NewGame{erGame})
		if err != nil || !slices.Equal(sum.Configured, []string{"ELDEN RING"}) {
			t.Errorf("%s: summary = %+v, %v", p, sum, err)
		}
	}
}

func TestAddGamesRefusesInvalidConfig(t *testing.T) {
	p := writeConfig(t, "games: [")
	_, err := AddGames(p, []NewGame{erGame})
	if err == nil || !strings.Contains(err.Error(), "config is invalid") {
		t.Fatalf("err = %v", err)
	}
	if readText(t, p) != "games: [" {
		t.Error("invalid config was modified")
	}
	if _, err := os.Stat(p + ".bak"); err == nil {
		t.Error(".bak written for refused merge")
	}
}
