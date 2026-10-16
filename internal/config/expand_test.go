package config

import "testing"

var testEnv EnvFunc = func(key string) string {
	return map[string]string{
		"APPDATA":     `C:\Users\A\AppData\Roaming`,
		"USERPROFILE": `C:\Users\A`,
	}[key]
}

func TestExpandPath(t *testing.T) {
	cases := map[string]string{
		"%APPDATA%/EldenRing": `C:\Users\A\AppData\Roaming\EldenRing`,
		"~/Saves/*.sav":       `C:\Users\A\Saves\*.sav`,
		"%NOPE%/x":            `%NOPE%\x`,
		`D:\Games\Save`:       `D:\Games\Save`,
	}
	for in, want := range cases {
		if got := ExpandPath(in, testEnv); got != want {
			t.Errorf("ExpandPath(%q) = %q, want %q", in, got, want)
		}
	}
}
