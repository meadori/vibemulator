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
		0xA5: {"LDA", c.lda, c.zp0, "zp0", 3},
		0x85: {"STA", c.sta, c.zp0, "zp0", 3},
		0xAA: {"TAX", c.tax, c.imp, "imp", 2},
	}
}

// Addressing Modes

func (c *CPU) imp() {
	c.fetched = c.A
}

func (c *CPU) imm() {
	c.addrAbs = c.PC
	c.PC++
}

func (c *CPU) zp0() {
	c.addrAbs = uint16(c.bus.Read(c.PC))
	c.PC++
}

func (c *CPU) zpx() {
	c.addrAbs = uint16(c.bus.Read(c.PC) + c.X)
	c.PC++
	c.addrAbs &= 0x00FF
}

func (c *CPU) zpy() {
	c.addrAbs = uint16(c.bus.Read(c.PC) + c.Y)
	c.PC++
	c.addrAbs &= 0x00FF
}

func (c *CPU) rel() {
	c.addrRel = uint16(c.bus.Read(c.PC))
	c.PC++
	if c.addrRel&0x80 != 0 {
		c.addrRel |= 0xFF00
	}
}

func (c *CPU) abs() {
	lo := uint16(c.bus.Read(c.PC))
	c.PC++
	hi := uint16(c.bus.Read(c.PC))
	c.PC++
	c.addrAbs = (hi << 8) | lo
}

func (c *CPU) abx() {
	lo := uint16(c.bus.Read(c.PC))
	c.PC++
	hi := uint16(c.bus.Read(c.PC))
	c.PC++
	c.addrAbs = (hi << 8) | lo
	c.addrAbs += uint16(c.X)

	if (c.addrAbs & 0xFF00) != (hi << 8) {
		c.cycles++
	}
}

func (c *CPU) aby() {
	lo := uint16(c.bus.Read(c.PC))
	c.PC++
	hi := uint16(c.bus.Read(c.PC))
	c.PC++
	c.addrAbs = (hi << 8) | lo
	c.addrAbs += uint16(c.Y)

	if (c.addrAbs & 0xFF00) != (hi << 8) {
		c.cycles++
	}
}

func (c *CPU) ind() {
	ptrLo := uint16(c.bus.Read(c.PC))
	c.PC++
	ptrHi := uint16(c.bus.Read(c.PC))
	c.PC++
	ptr := (ptrHi << 8) | ptrLo

	if ptrLo == 0x00FF { // Simulate page boundary hardware bug
		c.addrAbs = (uint16(c.bus.Read(ptr&0xFF00)) << 8) | uint16(c.bus.Read(ptr))
	} else {
		c.addrAbs = (uint16(c.bus.Read(ptr+1)) << 8) | uint16(c.bus.Read(ptr))
	}
}

func (c *CPU) izx() {
	t := uint16(c.bus.Read(c.PC))
	c.PC++
	lo := uint16(c.bus.Read((t + uint16(c.X)) & 0x00FF))
	hi := uint16(c.bus.Read((t + uint16(c.X) + 1) & 0x00FF))
	c.addrAbs = (hi << 8) | lo
}

func (c *CPU) izy() {
	t := uint16(c.bus.Read(c.PC))
	c.PC++
	lo := uint16(c.bus.Read(t & 0x00FF))
	hi := uint16(c.bus.Read((t + 1) & 0x00FF))
	c.addrAbs = (hi << 8) | lo
	c.addrAbs += uint16(c.Y)

	if (c.addrAbs & 0xFF00) != (hi << 8) {
		c.cycles++
	}
}

// Instructions

func (c *CPU) sta() {
	c.bus.Write(c.addrAbs, c.A)
}

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
