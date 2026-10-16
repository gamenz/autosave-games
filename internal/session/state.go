// Package session tracks one game's play session and decides when to back up.
package session

import "time"

// State is the session phase of one game.
type State int

const (
	Idle State = iota
	Running
	Settling
	WaitingStable
	BackingUp
)

func (s State) String() string {
	switch s {
	case Idle:
		return "Idle"
	case Running:
		return "Running"
	case Settling:
		return "Settling"
	case WaitingStable:
		return "WaitingStable"
	case BackingUp:
		return "BackingUp"
	}
	return "Unknown"
}

// Action is what the caller must do after a Step.
type Action int

const (
	None Action = iota
	SnapshotPre
	BackupPost
)

func (a Action) String() string {
	switch a {
	case SnapshotPre:
		return "SnapshotPre"
	case BackupPost:
		return "BackupPost"
	}
	return "None"
}

const (
	// StableInterval is the spacing between save-file stability checks.
	StableInterval = 2 * time.Second
	// StableTimeout forces the post backup when files never settle.
	StableTimeout = 60 * time.Second
)

// Input is one observation fed to the machine.
type Input struct {
	Now     time.Time
	Running bool
	Stable  bool
}

// Machine is the pure per-game session state machine (spec §4).
type Machine struct {
	state    State
	settle   time.Duration
	markedAt time.Time
}

// NewMachine starts in Running (without a pre snapshot) when the game is
// already running at app start, otherwise in Idle.
func NewMachine(settle time.Duration, alreadyRunning bool) *Machine {
	m := &Machine{state: Idle, settle: settle}
	if alreadyRunning {
		m.state = Running
	}
	return m
}

// State returns the current state.
func (m *Machine) State() State { return m.state }

// Step advances the machine by one observation.
func (m *Machine) Step(in Input) Action {
	switch m.state {
	case Idle:
		if in.Running {
			m.state = Running
			return SnapshotPre
		}
	case Running:
		if !in.Running {
			m.state, m.markedAt = Settling, in.Now
		}
	case Settling:
		if in.Running {
			m.state = Running
		} else if in.Now.Sub(m.markedAt) >= m.settle {
			m.state, m.markedAt = WaitingStable, in.Now
		}
	case WaitingStable:
		if in.Running {
			m.state = Running
		} else if in.Stable || in.Now.Sub(m.markedAt) >= StableTimeout {
			m.state = BackingUp
			return BackupPost
		}
	}
	return None
}

// BackupDone ends the BackingUp phase.
func (m *Machine) BackupDone() {
	if m.state == BackingUp {
		m.state = Idle
	}
}
