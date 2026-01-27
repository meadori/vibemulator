package cpu

import (
	"fmt"
)

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
	Cycles  int // Exported
	Lookup  [256]Instruction

	fetched uint8
	addrAbs uint16
	addrRel uint16
}


// New creates a new CPU instance.
func New() *CPU {
	c := &CPU{}
	c.Lookup = c.createLookupTable()
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

	c.Cycles = 8 // Updated
}

// NMI is a non-maskable interrupt.
func (c *CPU) NMI() {
	c.push(byte((c.PC >> 8) & 0x00FF))
	c.push(byte(c.PC & 0x00FF))

	c.setFlag(B, false)
	c.setFlag(U, true)
	c.setFlag(I, true)
	c.push(c.P)

	c.addrAbs = 0xFFFA
	lo := uint16(c.bus.Read(c.addrAbs))
	hi := uint16(c.bus.Read(c.addrAbs + 1))
	c.PC = (hi << 8) | lo

	c.Cycles = 8 // Updated
}

// LogState prints the current CPU state in a nestest-like format.
func (c *CPU) LogState() string {
	// PPU cycle count and total cycles are omitted for now as they are not directly available in CPU struct.
	// P-register flags are displayed as a hex value.
	return fmt.Sprintf("%04X A:%02X X:%02X Y:%02X P:%02X SP:%02X",
		c.PC, c.A, c.X, c.Y, c.P, c.SP)
}


// Clock performs one clock cycle.
func (c *CPU) Clock() {
	if c.Cycles == 0 { // Updated
		c.opcode = c.bus.Read(c.PC)
		c.PC++

		instr := c.Lookup[c.opcode]
		if instr.Operate == nil {
			// Invalid instruction
			return
		}
		c.Cycles = instr.Cycles // Updated
		instr.AddrMode()
		instr.Operate()
	}
	c.Cycles-- // Updated
}
