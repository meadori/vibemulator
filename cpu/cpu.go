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

	opcode  byte
	cycles  int
	lookup  [256]Instruction

	fetched uint8
	addrAbs uint16
	addrRel uint16
}


// New creates a new CPU instance.
func New() *CPU {
	c := &CPU{}
	c.lookup = c.createLookupTable()
	return c
}

// ConnectBus connects the CPU to the bus.
func (c *CPU) ConnectBus(bus Bus) {
	c.bus = bus
}

// Reset resets the CPU to its initial state.
func (c *CPU) Reset() {
	c.addrAbs = 0xFFFC
	lo := uint16(c.bus.Read(c.addrAbs))
	hi := uint16(c.bus.Read(c.addrAbs + 1))
	c.PC = (hi << 8) | lo

	c.A = 0
	c.X = 0
	c.Y = 0
	c.SP = 0xFD
	c.P = 0x00 | U

	c.cycles = 8
}

// Clock performs one clock cycle.
func (c *CPU) Clock() {
	if c.cycles == 0 {
		c.opcode = c.bus.Read(c.PC)
		c.PC++

		instr := c.lookup[c.opcode]
		if instr.Operate == nil {
			// Invalid instruction
			return
		}
		c.cycles = instr.Cycles
		instr.AddrMode()
		instr.Operate()
	}
	c.cycles--
}

