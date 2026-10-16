package session

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 17, 20, 0, 0, 0, time.Local)

func at(sec int) time.Time { return t0.Add(time.Duration(sec) * time.Second) }

type step struct {
	sec             int
	running, stable bool
	want            Action
	state           State
}

func run(t *testing.T, m *Machine, steps []step) {
	t.Helper()
	for i, s := range steps {
		got := m.Step(Input{Now: at(s.sec), Running: s.running, Stable: s.stable})
		if got != s.want || m.State() != s.state {
			t.Fatalf("step %d (t=%ds): got %v/%v want %v/%v", i, s.sec, got, m.State(), s.want, s.state)
		}
	}
}

const settle = 15 * time.Second

func TestNormalSession(t *testing.T) {
	m := NewMachine(settle, false)
	run(t, m, []step{
		{0, false, false, None, Idle},
		{3, true, false, SnapshotPre, Running},
		{6, true, false, None, Running},
		{9, false, false, None, Settling},
		{21, false, false, None, Settling},
		{24, false, false, None, WaitingStable},
		{26, false, true, BackupPost, BackingUp},
		{28, true, false, None, BackingUp},
	})
	m.BackupDone()
	if m.State() != Idle {
		t.Fatalf("after BackupDone state = %v", m.State())
	}
}

func TestFlickerDuringSettleNoExtraPre(t *testing.T) {
	run(t, NewMachine(settle, false), []step{
		{0, true, false, SnapshotPre, Running},
		{3, false, false, None, Settling},
		{6, true, false, None, Running},
		{9, false, false, None, Settling},
		{21, false, false, None, Settling},
		{24, false, false, None, WaitingStable},
	})
}

func TestAlreadyRunningAtStartup(t *testing.T) {
	m := NewMachine(settle, true)
	if m.State() != Running {
		t.Fatalf("state = %v", m.State())
	}
	run(t, m, []step{{0, true, false, None, Running}})
}

func TestStableTimeout(t *testing.T) {
	run(t, NewMachine(settle, false), []step{
		{0, true, false, SnapshotPre, Running},
		{3, false, false, None, Settling},
		{18, false, false, None, WaitingStable},
		{20, false, false, None, WaitingStable},
		{50, false, false, None, WaitingStable},
		{77, false, false, None, WaitingStable},
		{78, false, false, BackupPost, BackingUp},
	})
}

func TestStableIgnoredOutsideWaiting(t *testing.T) {
	run(t, NewMachine(settle, false), []step{
		{0, true, true, SnapshotPre, Running},
		{3, false, true, None, Settling},
	})
}

func TestRestartDuringWaitingStable(t *testing.T) {
	run(t, NewMachine(settle, false), []step{
		{0, true, false, SnapshotPre, Running},
		{3, false, false, None, Settling},
		{18, false, false, None, WaitingStable},
		{20, true, false, None, Running},
	})
}

func TestSecondSessionAfterBackup(t *testing.T) {
	m := NewMachine(settle, false)
	run(t, m, []step{
		{0, true, false, SnapshotPre, Running},
		{3, false, false, None, Settling},
		{18, false, false, None, WaitingStable},
		{20, false, true, BackupPost, BackingUp},
	})
	m.BackupDone()
	run(t, m, []step{{30, true, false, SnapshotPre, Running}})
}
