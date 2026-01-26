package bus

import "github.com/meadori/vibemulator/cpu"

// Bus represents the system bus.
type Bus struct {
	cpu *cpu.CPU
}

// New creates a new Bus instance.
func New() *Bus {
	return &Bus{
		cpu: cpu.New(),
	}
}
