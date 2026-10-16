package tray

import (
	"fmt"
	"os"
	"testing"
)

func TestSingleInstance(t *testing.T) {
	name := fmt.Sprintf(`Local\autosave-games-test-%d`, os.Getpid())
	release, already, err := AcquireSingleInstance(name)
	if err != nil || already {
		t.Fatalf("first acquire: already=%v err=%v", already, err)
	}
	release2, already2, err := AcquireSingleInstance(name)
	if err != nil || !already2 {
		t.Fatalf("second acquire: already=%v err=%v", already2, err)
	}
	release2()
	release()
}
