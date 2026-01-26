package cpu

// Instruction represents a 6502 instruction.
type Instruction struct {
	Name         string
	Operate      func()
	AddrMode     func()
	AddrModeName string
	Cycles       int
}

func (c *CPU) createLookupTable() [256]Instruction {
	return [256]Instruction{
		0xA9: {"LDA", c.lda, c.imm, "imm", 2},
		0xAA: {"TAX", c.tax, c.imp, "imp", 2},
	}
}

// Addressing Modes

func (c *CPU) imp() {
	// Do nothing
}

func (c *CPU) imm() {
	c.addrAbs = c.PC
	c.PC++
}

// Instructions

func (c *CPU) tax() {
	c.X = c.A
	c.setFlag('Z', c.X == 0)
	c.setFlag('N', c.X&0x80 != 0)
}

func (c *CPU) lda() {
	c.fetch()
	c.A = c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
}

func (c *CPU) fetch() {
	if c.lookup[c.opcode].AddrModeName != "imp" {
		c.fetched = c.bus.Read(c.addrAbs)
	}
}

// Flags

func (c *CPU) setFlag(flag byte, v bool) {
	if v {
		c.P |= 1 << Flag(flag)
	} else {
		c.P &= ^(1 << Flag(flag))
	}
}

func (c *CPU) getFlag(flag byte) byte {
	if (c.P & (1 << Flag(flag))) > 0 {
		return 1
	}
	return 0
}

const (
	C byte = (1 << 0) // Carry Bit
	Z byte = (1 << 1) // Zero
	I byte = (1 << 2) // Disable Interrupts
	D byte = (1 << 3) // Decimal Mode
	B byte = (1 << 4) // Break
	U byte = (1 << 5) // Unused
	V byte = (1 << 6) // Overflow
	N byte = (1 << 7) // Negative
)

func Flag(flag byte) byte {
	switch flag {
	case 'C':
		return 0
	case 'Z':
		return 1
	case 'I':
		return 2
	case 'D':
		return 3
	case 'B':
		return 4
	case 'U':
		return 5
	case 'V':
		return 6
	case 'N':
		return 7
	}
	return 0
}
