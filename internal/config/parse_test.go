package config

import "testing"

const sampleYAML = `
backup_root: "~/Backups"
games:
  - {name: Elden Ring, processes: [EldenRing.exe], saves: ["%APPDATA%/EldenRing"]}
  - {name: HK, processes: [hk.exe], saves: [x], keep_last: 50, settle_seconds: 5}
`

func TestParseDefaultsAndOverrides(t *testing.T) {
	c, err := Parse([]byte(sampleYAML), testEnv)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if c.PollSeconds != 3 || c.Notify != NotifyErrors {
		t.Errorf("defaults: poll=%d notify=%q", c.PollSeconds, c.Notify)
	}
	if c.BackupRoot != `C:\Users\A\Backups` {
		t.Errorf("BackupRoot = %q", c.BackupRoot)
	}
	g := c.Games[0]
	if g.Slug != "elden-ring" || g.KeepLast != 20 || g.SettleSeconds != 5 {
		t.Errorf("game 0 = %+v", g)
	}
	if g.Saves[0] != `C:\Users\A\AppData\Roaming\EldenRing` {
		t.Errorf("Saves[0] = %q", g.Saves[0])
	}
	if g.Processes[0] != "eldenring.exe" {
		t.Errorf("Processes[0] = %q, want lower-cased", g.Processes[0])
	}
	if h := c.Games[1]; h.KeepLast != 50 || h.SettleSeconds != 5 {
		t.Errorf("game 1 overrides = %+v", h)
	}
}

func TestParseInvalidYAML(t *testing.T) {
	if _, err := Parse([]byte("games: [unclosed"), testEnv); err == nil {
		t.Fatal("expected YAML error")
	}
}

func TestFindGame(t *testing.T) {
	c, err := Parse([]byte(sampleYAML), testEnv)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"ELDEN RING", "elden-ring"} {
		if g, ok := c.FindGame(q); !ok || g.Name != "Elden Ring" {
			t.Errorf("FindGame(%q) = %v, %v", q, g, ok)
		}
	}
	if _, ok := c.FindGame("zelda"); ok {
		t.Error("FindGame(zelda) should not match")
	}
}
