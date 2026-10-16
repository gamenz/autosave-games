package config

import (
	"strings"
	"testing"
)

func TestSplitGlobBase(t *testing.T) {
	cases := []struct{ in, base, pat string }{
		{`C:\S\Hollow Knight\user*.dat`, `C:\S\Hollow Knight`, `user*.dat`},
		{`C:\S\slot?\data`, `C:\S`, `slot?\data`},
		{`C:\S\EldenRing`, `C:\S\EldenRing`, ``},
		{`C:\S\[ab]\x`, `C:\S`, `[ab]\x`},
	}
	for _, tc := range cases {
		base, pat := SplitGlobBase(tc.in)
		if base != tc.base || pat != tc.pat {
			t.Errorf("SplitGlobBase(%q) = (%q, %q), want (%q, %q)", tc.in, base, pat, tc.base, tc.pat)
		}
	}
}

func TestResolveSavesIndexes(t *testing.T) {
	got := ResolveSaves([]string{`C:\A`, `C:\B\*.sav`})
	if len(got) != 2 || got[0].Index != 0 || got[1].Index != 1 {
		t.Fatalf("ResolveSaves = %+v", got)
	}
	if got[1].AbsBase != `C:\B` || got[1].Pattern != `*.sav` {
		t.Errorf("second base = %+v", got[1])
	}
}

func TestIsWithin(t *testing.T) {
	cases := []struct {
		child, parent string
		want          bool
	}{
		{`D:\Saves\Backups`, `d:\saves`, true},
		{`D:\Saves`, `D:\Saves`, true},
		{`D:\SavesX`, `D:\Saves`, false},
		{`D:\Other`, `D:\Saves`, false},
	}
	for _, tc := range cases {
		if got := IsWithin(tc.child, tc.parent); got != tc.want {
			t.Errorf("IsWithin(%q, %q) = %v, want %v", tc.child, tc.parent, got, tc.want)
		}
	}
}

func TestValidateBackupRootOverlap(t *testing.T) {
	cases := []string{
		"backup_root: D:/Saves/Backups\ngames: [{name: A, processes: [a.exe], saves: [D:/Saves]}]",
		"backup_root: D:/B\ngames: [{name: A, processes: [a.exe], saves: [D:/B/x/*.sav]}]",
	}
	for _, yaml := range cases {
		err := Validate(mustParse(t, yaml))
		want := `game "A": saves[0]: overlaps backup_root`
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Validate(%q) = %v, want %q", yaml, err, want)
		}
	}
}
