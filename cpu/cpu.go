package cpu

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
}

// New creates a new CPU instance.
func New() *CPU {
	return &CPU{}
}
