package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupWritesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	log, closer, err := Setup(dir)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	log.Info("hello", "game", "Test")
	closer.Close()
	b, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "hello") {
		t.Errorf("log content = %q", b)
	}
}
