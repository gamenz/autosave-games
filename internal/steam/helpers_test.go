package steam

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// acf renders a minimal appmanifest.
func acf(appid int, name, installdir string) string {
	return fmt.Sprintf("\"AppState\"\n{\n\t\"appid\"\t\t\"%d\"\n\t\"name\"\t\t\"%s\"\n\t\"installdir\"\t\t\"%s\"\n}\n",
		appid, name, installdir)
}

// vdfPath escapes a Windows path for a VDF string.
func vdfPath(p string) string { return strings.ReplaceAll(p, `\`, `\\`) }
