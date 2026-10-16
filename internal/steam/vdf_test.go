package steam

import (
	"strings"
	"testing"
)

const sampleVDF = `"libraryfolders"
{
	// comment
	"0"
	{
		"path"		"D:\\Steam"
		"apps" { "1245620" "71248674974" }
	}
	"1" { "path" "E:\\Trò chơi\\SteamLibrary" }
}`

func TestParseVDF(t *testing.T) {
	root, err := ParseVDF(strings.NewReader(sampleVDF))
	if err != nil {
		t.Fatal(err)
	}
	lf := root.Child("LibraryFolders")
	if lf == nil || !lf.Block {
		t.Fatalf("libraryfolders block missing: %+v", root)
	}
	if got := lf.Child("0").Get("path"); got != `D:\Steam` {
		t.Errorf("path 0 = %q", got)
	}
	if got := lf.Child("0").Child("apps").Get("1245620"); got != "71248674974" {
		t.Errorf("apps = %q", got)
	}
	if got := lf.Child("1").Get("path"); got != `E:\Trò chơi\SteamLibrary` {
		t.Errorf("path 1 = %q", got)
	}
	if lf.Child("nope") != nil || lf.Get("nope") != "" {
		t.Error("missing key should give nil / empty")
	}
}

func TestParseVDFEscapes(t *testing.T) {
	root, err := ParseVDF(strings.NewReader(`"k" "a \"b\" \t c"` + "\n" + `bare value_1`))
	if err != nil {
		t.Fatal(err)
	}
	if got := root.Get("k"); got != "a \"b\" \t c" {
		t.Errorf("k = %q", got)
	}
	if got := root.Get("bare"); got != "value_1" {
		t.Errorf("bare = %q", got)
	}
}

func TestParseVDFErrors(t *testing.T) {
	for _, bad := range []string{`"k" "unterminated`, `"a" { "b" "c"`, `}`, `"a"`} {
		if _, err := ParseVDF(strings.NewReader(bad)); err == nil || !strings.Contains(err.Error(), "line") {
			t.Errorf("ParseVDF(%q) err = %v, want line-numbered error", bad, err)
		}
	}
}
