package cpu

// Bus defines the interface for the CPU to interact with the bus.
type Bus interface {
	Read(addr uint16) byte
	Write(addr uint16, data byte)
}

// CPU represents the 6502 CPU.
type CPU struct {
	// Program Counter
	PC uint16

	// Stack Pointer
	SP byte

	// Accumulator
	A byte

	// Index Register X
	X byte

	// Index Register Y
	Y byte

	// Processor Status
	P byte

	bus Bus
}

// New creates a new CPU instance.
func New() *CPU {
	return &CPU{}
}

// ConnectBus connects the CPU to the bus.
func (c *CPU) ConnectBus(bus Bus) {
	c.bus = bus
}
