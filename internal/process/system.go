package process

import (
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

type systemLister struct{}

// NewSystemLister lists real processes via gopsutil.
func NewSystemLister() Lister { return systemLister{} }

func (systemLister) Names() (map[string]bool, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}
	names := make(map[string]bool, len(procs))
	for _, p := range procs {
		// Processes may exit or deny access between listing and Name().
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}
		names[strings.ToLower(name)] = true
	}
	return names, nil
}
