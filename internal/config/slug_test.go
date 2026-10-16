package config

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Elden Ring":         "elden-ring",
		"  Hollow  Knight! ": "hollow-knight",
		"NieR:Automata":      "nier-automata",
		"Đế Chế 2":           "đế-chế-2",
		"!!!":                "",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
