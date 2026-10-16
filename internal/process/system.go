package process

import (
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

type systemLister struct{}

// NewSystemLister lists real processes via gopsutil.
func NewSystemLister() Lister { return systemLister{} }

func (systemLister) Snapshot() (Snapshot, error) {
	procs, err := process.Processes()
	if err != nil {
		return Snapshot{}, fmt.Errorf("list processes: %w", err)
	}
	snap := Snapshot{Names: make(map[string]bool, len(procs))}
	for _, p := range procs {
		// Processes may exit or deny access between listing and these calls.
		if name, err := p.Name(); err == nil && name != "" {
			snap.Names[strings.ToLower(name)] = true
		}
		if exe, err := p.Exe(); err == nil && exe != "" {
			snap.Paths = append(snap.Paths, strings.ToLower(exe))
		}
	}
	return snap, nil
}
