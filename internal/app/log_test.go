package app

import (
	"bytes"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// syncBuffer is a goroutine-safe log sink for tests.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestLogsStateTransitions(t *testing.T) {
	h := newHarness(t, "", nil)
	h.tick(0)
	h.fake.SetRunning("game.exe")
	h.tick(3)
	writeFile(t, filepath.Join(h.saves, "a.sav"), "v2")
	h.fake.SetRunning()
	for i := 0; i < 6; i++ {
		h.tick(3)
	}
	h.tick(2)
	h.tick(2)
	h.tick(0)

	logs := h.logs.String()
	for _, want := range []string{
		`msg="session state" game="Test Game" from=Idle to=Running`,
		`msg="session state" game="Test Game" from=Running to=Settling`,
		`msg="session state" game="Test Game" from=Settling to=WaitingStable`,
		`msg="session state" game="Test Game" from=WaitingStable to=BackingUp stable=true`,
		`msg="session state" game="Test Game" from=BackingUp to=Idle`,
	} {
		if !strings.Contains(logs, want) {
			t.Errorf("log missing %q\n%s", want, logs)
		}
	}
}
