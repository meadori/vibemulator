package bus

import "github.com/meadori/vibemulator/cpu"

// Bus represents the system bus.
type Bus struct {
	cpu *cpu.CPU
	ram [2048]byte
}

// New creates a new Bus instance.
func New() *Bus {
	bus := &Bus{
		cpu: cpu.New(),
	}
	bus.cpu.ConnectBus(bus)
	return bus
}

// Read reads a byte from the bus.
func (b *Bus) Read(addr uint16) byte {
	if addr >= 0x0000 && addr <= 0x1FFF {
		return b.ram[addr&0x07FF]
	}
	return 0
}

// Write writes a byte to the bus.
func (b *Bus) Write(addr uint16, data byte) {
	if addr >= 0x0000 && addr <= 0x1FFF {
		b.ram[addr&0x07FF] = data
	}
}
