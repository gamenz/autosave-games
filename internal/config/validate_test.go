package config

import (
	"strings"
	"testing"
)

const rootLine = "backup_root: r\n"

var validateCases = []struct{ name, yaml, wantErr string }{
	{"no root", `games: [{name: A, processes: [a.exe], saves: [x]}]`, "backup_root: required"},
	{"no name", rootLine + `games: [{processes: [a.exe], saves: [x]}]`, "game #1: name: required"},
	{"no process", rootLine + `games: [{name: A, saves: [x]}]`, `game "A": processes or install_dir: at least one required`},
	{"no saves", rootLine + `games: [{name: A, processes: [a.exe]}]`, `game "A": saves: at least one required`},
	{"dup name", rootLine + `games: [{name: A, processes: [a.exe], saves: [x]}, {name: a, processes: [b.exe], saves: [y]}]`, `game "a": duplicate name`},
	{"dup slug", rootLine + `games: [{name: A B, processes: [a.exe], saves: [x]}, {name: a-b, processes: [b.exe], saves: [y]}]`, `duplicate slug "a-b"`},
	{"empty slug", rootLine + `games: [{name: '!!', processes: [a.exe], saves: [x]}]`, `game "!!": name: no letters or digits`},
	{"notify", rootLine + `notify: loud`, "notify: must be one of all, errors, none"},
	{"negative keep", rootLine + `games: [{name: A, processes: [a.exe], saves: [x], keep_last: -1}]`, `game "A": keep_last: must be > 0`},
	{"negative poll", rootLine + `poll_seconds: -2`, "poll_seconds: must be > 0"},
}

func mustParse(t *testing.T, yaml string) *Config {
	t.Helper()
	c, err := Parse([]byte(yaml), testEnv)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return c
}

func TestValidateRules(t *testing.T) {
	for _, tc := range validateCases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(mustParse(t, tc.yaml))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidateOK(t *testing.T) {
	if err := Validate(mustParse(t, sampleYAML)); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateReportsAll(t *testing.T) {
	c := mustParse(t, `notify: loud
games: [{name: A, saves: [x]}]`)
	err := Validate(c)
	if err == nil {
		t.Fatal("expected errors")
	}
	for _, want := range []string{"backup_root", "notify", "processes or install_dir"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err, want)
		}
	}
}
