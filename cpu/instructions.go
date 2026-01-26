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
		// LDA
		0xA9: {"LDA", c.lda, c.imm, "imm", 2},
		0xA5: {"LDA", c.lda, c.zp0, "zp0", 3},
		0xB5: {"LDA", c.lda, c.zpx, "zpx", 4},
		0xAD: {"LDA", c.lda, c.abs, "abs", 4},
		0xBD: {"LDA", c.lda, c.abx, "abx", 4},
		0xB9: {"LDA", c.lda, c.aby, "aby", 4},
		0xA1: {"LDA", c.lda, c.izx, "izx", 6},
		0xB1: {"LDA", c.lda, c.izy, "izy", 5},

		// LDX
		0xA2: {"LDX", c.ldx, c.imm, "imm", 2},
		0xA6: {"LDX", c.ldx, c.zp0, "zp0", 3},
		0xB6: {"LDX", c.ldx, c.zpy, "zpy", 4},
		0xAE: {"LDX", c.ldx, c.abs, "abs", 4},
		0xBE: {"LDX", c.ldx, c.aby, "aby", 4},

		// LDY
		0xA0: {"LDY", c.ldy, c.imm, "imm", 2},
		0xA4: {"LDY", c.ldy, c.zp0, "zp0", 3},
		0xB4: {"LDY", c.ldy, c.zpx, "zpx", 4},
		0xAC: {"LDY", c.ldy, c.abs, "abs", 4},
		0xBC: {"LDY", c.ldy, c.abx, "abx", 4},

		// STA
		0x85: {"STA", c.sta, c.zp0, "zp0", 3},
		0x95: {"STA", c.sta, c.zpx, "zpx", 4},
		0x8D: {"STA", c.sta, c.abs, "abs", 4},
		0x9D: {"STA", c.sta, c.abx, "abx", 5},
		0x99: {"STA", c.sta, c.aby, "aby", 5},
		0x81: {"STA", c.sta, c.izx, "izx", 6},
		0x91: {"STA", c.sta, c.izy, "izy", 6},

		// STX
		0x86: {"STX", c.stx, c.zp0, "zp0", 3},
		0x96: {"STX", c.stx, c.zpy, "zpy", 4},
		0x8E: {"STX", c.stx, c.abs, "abs", 4},

		// STY
		0x84: {"STY", c.sty, c.zp0, "zp0", 3},
		0x94: {"STY", c.sty, c.zpx, "zpx", 4},
		0x8C: {"STY", c.sty, c.abs, "abs", 4},

		// Arithmetic
		0x69: {"ADC", c.adc, c.imm, "imm", 2},
		0x65: {"ADC", c.adc, c.zp0, "zp0", 3},
		0x75: {"ADC", c.adc, c.zpx, "zpx", 4},
		0x6D: {"ADC", c.adc, c.abs, "abs", 4},
		0x7D: {"ADC", c.adc, c.abx, "abx", 4},
		0x79: {"ADC", c.adc, c.aby, "aby", 4},
		0x61: {"ADC", c.adc, c.izx, "izx", 6},
		0x71: {"ADC", c.adc, c.izy, "izy", 5},
		0xE9: {"SBC", c.sbc, c.imm, "imm", 2},
		0xE5: {"SBC", c.sbc, c.zp0, "zp0", 3},
		0xF5: {"SBC", c.sbc, c.zpx, "zpx", 4},
		0xED: {"SBC", c.sbc, c.abs, "abs", 4},
		0xFD: {"SBC", c.sbc, c.abx, "abx", 4},
		0xF9: {"SBC", c.sbc, c.aby, "aby", 4},
		0xE1: {"SBC", c.sbc, c.izx, "izx", 6},
		0xF1: {"SBC", c.sbc, c.izy, "izy", 5},

		// Increment/Decrement
		0xE6: {"INC", c.inc, c.zp0, "zp0", 5},
		0xF6: {"INC", c.inc, c.zpx, "zpx", 6},
		0xEE: {"INC", c.inc, c.abs, "abs", 6},
		0xFE: {"INC", c.inc, c.abx, "abx", 7},
		0xE8: {"INX", c.inx, c.imp, "imp", 2},
		0xC8: {"INY", c.iny, c.imp, "imp", 2},
		0xC6: {"DEC", c.dec, c.zp0, "zp0", 5},
		0xD6: {"DEC", c.dec, c.zpx, "zpx", 6},
		0xCE: {"DEC", c.dec, c.abs, "abs", 6},
		0xDE: {"DEC", c.dec, c.abx, "abx", 7},
		0xCA: {"DEX", c.dex, c.imp, "imp", 2},
		0x88: {"DEY", c.dey, c.imp, "imp", 2},

		// Logical
		0x29: {"AND", c.and, c.imm, "imm", 2},
		0x25: {"AND", c.and, c.zp0, "zp0", 3},
		0x35: {"AND", c.and, c.zpx, "zpx", 4},
		0x2D: {"AND", c.and, c.abs, "abs", 4},
		0x3D: {"AND", c.and, c.abx, "abx", 4},
		0x39: {"AND", c.and, c.aby, "aby", 4},
		0x21: {"AND", c.and, c.izx, "izx", 6},
		0x31: {"AND", c.and, c.izy, "izy", 5},
		0x09: {"ORA", c.ora, c.imm, "imm", 2},
		0x05: {"ORA", c.ora, c.zp0, "zp0", 3},
		0x15: {"ORA", c.ora, c.zpx, "zpx", 4},
		0x0D: {"ORA", c.ora, c.abs, "abs", 4},
		0x1D: {"ORA", c.ora, c.abx, "abx", 4},
		0x19: {"ORA", c.ora, c.aby, "aby", 4},
		0x01: {"ORA", c.ora, c.izx, "izx", 6},
		0x11: {"ORA", c.ora, c.izy, "izy", 5},
		0x49: {"EOR", c.eor, c.imm, "imm", 2},
		0x45: {"EOR", c.eor, c.zp0, "zp0", 3},
		0x55: {"EOR", c.eor, c.zpx, "zpx", 4},
		0x4D: {"EOR", c.eor, c.abs, "abs", 4},
		0x5D: {"EOR", c.eor, c.abx, "abx", 4},
		0x59: {"EOR", c.eor, c.aby, "aby", 4},
		0x41: {"EOR", c.eor, c.izx, "izx", 6},
		0x51: {"EOR", c.eor, c.izy, "izy", 5},

		// Shift/Rotate
		0x0A: {"ASL", c.asl, c.imp, "imp", 2},
		0x06: {"ASL", c.asl, c.zp0, "zp0", 5},
		0x16: {"ASL", c.asl, c.zpx, "zpx", 6},
		0x0E: {"ASL", c.asl, c.abs, "abs", 6},
		0x1E: {"ASL", c.asl, c.abx, "abx", 7},
		0x4A: {"LSR", c.lsr, c.imp, "imp", 2},
		0x46: {"LSR", c.lsr, c.zp0, "zp0", 5},
		0x56: {"LSR", c.lsr, c.zpx, "zpx", 6},
		0x4E: {"LSR", c.lsr, c.abs, "abs", 6},
		0x5E: {"LSR", c.lsr, c.abx, "abx", 7},
		0x2A: {"ROL", c.rol, c.imp, "imp", 2},
		0x26: {"ROL", c.rol, c.zp0, "zp0", 5},
		0x36: {"ROL", c.rol, c.zpx, "zpx", 6},
		0x2E: {"ROL", c.rol, c.abs, "abs", 6},
		0x3E: {"ROL", c.rol, c.abx, "abx", 7},
		0x6A: {"ROR", c.ror, c.imp, "imp", 2},
		0x66: {"ROR", c.ror, c.zp0, "zp0", 5},
		0x76: {"ROR", c.ror, c.zpx, "zpx", 6},
		0x6E: {"ROR", c.ror, c.abs, "abs", 6},
		0x7E: {"ROR", c.ror, c.abx, "abx", 7},

		// Branch
		0x90: {"BCC", c.bcc, c.rel, "rel", 2},
		0xB0: {"BCS", c.bcs, c.rel, "rel", 2},
		0xF0: {"BEQ", c.beq, c.rel, "rel", 2},
		0x30: {"BMI", c.bmi, c.rel, "rel", 2},
		0xD0: {"BNE", c.bne, c.rel, "rel", 2},
		0x10: {"BPL", c.bpl, c.rel, "rel", 2},
		0x50: {"BVC", c.bvc, c.rel, "rel", 2},
		0x70: {"BVS", c.bvs, c.rel, "rel", 2},

		// Flags
		0x18: {"CLC", c.clc, c.imp, "imp", 2},
		0xD8: {"CLD", c.cld, c.imp, "imp", 2},
		0x58: {"CLI", c.cli, c.imp, "imp", 2},
		0xB8: {"CLV", c.clv, c.imp, "imp", 2},
		0x38: {"SEC", c.sec, c.imp, "imp", 2},
		0xF8: {"SED", c.sed, c.imp, "imp", 2},
		0x78: {"SEI", c.sei, c.imp, "imp", 2},

		// Compare
		0xC9: {"CMP", c.cmp, c.imm, "imm", 2},
		0xC5: {"CMP", c.cmp, c.zp0, "zp0", 3},
		0xD5: {"CMP", c.cmp, c.zpx, "zpx", 4},
		0xCD: {"CMP", c.cmp, c.abs, "abs", 4},
		0xDD: {"CMP", c.cmp, c.abx, "abx", 4},
		0xD9: {"CMP", c.cmp, c.aby, "aby", 4},
		0xC1: {"CMP", c.cmp, c.izx, "izx", 6},
		0xD1: {"CMP", c.cmp, c.izy, "izy", 5},
		0xE0: {"CPX", c.cpx, c.imm, "imm", 2},
		0xE4: {"CPX", c.cpx, c.zp0, "zp0", 3},
		0xEC: {"CPX", c.cpx, c.abs, "abs", 4},
		0xC0: {"CPY", c.cpy, c.imm, "imm", 2},
		0xC4: {"CPY", c.cpy, c.zp0, "zp0", 3},
		0xCC: {"CPY", c.cpy, c.abs, "abs", 4},

		// Jump
		0x4C: {"JMP", c.jmp, c.abs, "abs", 3},
		0x6C: {"JMP", c.jmp, c.ind, "ind", 5},
		0x20: {"JSR", c.jsr, c.abs, "abs", 6},
		0x60: {"RTS", c.rts, c.imp, "imp", 6},
		0x40: {"RTI", c.rti, c.imp, "imp", 6},

		// Other
		0x24: {"BIT", c.bit, c.zp0, "zp0", 3},
		0x2C: {"BIT", c.bit, c.abs, "abs", 4},
		0xEA: {"NOP", c.nop, c.imp, "imp", 2},

		// Stack
		0x48: {"PHA", c.pha, c.imp, "imp", 3},
		0x68: {"PLA", c.pla, c.imp, "imp", 4},
		0x08: {"PHP", c.php, c.imp, "imp", 3},
		0x28: {"PLP", c.plp, c.imp, "imp", 4},

		// Transfer
		0xAA: {"TAX", c.tax, c.imp, "imp", 2},
		0x8A: {"TXA", c.txa, c.imp, "imp", 2},
		0xA8: {"TAY", c.tay, c.imp, "imp", 2},
		0x98: {"TYA", c.tya, c.imp, "imp", 2},
		0xBA: {"TSX", c.tsx, c.imp, "imp", 2},
		0x9A: {"TXS", c.txs, c.imp, "imp", 2},
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

func (c *CPU) ldy() {
	c.fetch()
	c.Y = c.fetched
	c.setFlag('Z', c.Y == 0)
	c.setFlag('N', c.Y&0x80 != 0)
}

func (c *CPU) ldx() {
	c.fetch()
	c.X = c.fetched
	c.setFlag('Z', c.X == 0)
	c.setFlag('N', c.X&0x80 != 0)
}

func (c *CPU) sty() {
	c.bus.Write(c.addrAbs, c.Y)
}

func (c *CPU) stx() {
	c.bus.Write(c.addrAbs, c.X)
}

func (c *CPU) sta() {
	c.bus.Write(c.addrAbs, c.A)
}

func (c *CPU) plp() {
	c.P = c.pop()
	c.setFlag(U, true)
}

func (c *CPU) php() {
	c.push(c.P | B | U)
	c.setFlag(B, false)
	c.setFlag(U, false)
}

func (c *CPU) pla() {
	c.A = c.pop()
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
}

func (c *CPU) pha() {
	c.push(c.A)
}

func (c *CPU) push(data byte) {
	c.bus.Write(0x0100+uint16(c.SP), data)
	c.SP--
}

func (c *CPU) pop() byte {
	c.SP++
	return c.bus.Read(0x0100 + uint16(c.SP))
}

func (c *CPU) tya() {
	c.A = c.Y
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
}

func (c *CPU) tay() {
	c.Y = c.A
	c.setFlag('Z', c.Y == 0)
	c.setFlag('N', c.Y&0x80 != 0)
}

func (c *CPU) txa() {
	c.A = c.X
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
}

func (c *CPU) tsx() {
	c.X = c.SP
	c.setFlag('Z', c.X == 0)
	c.setFlag('N', c.X&0x80 != 0)
}

func (c *CPU) txs() {
	c.SP = c.X
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

func (c *CPU) sbc() {
	c.fetch()
	temp := uint16(c.A) - uint16(c.fetched) - (1 - uint16(c.getFlag('C')))
	c.setFlag('C', temp < 0x100)
	c.setFlag('Z', (temp&0x00FF) == 0)
	c.setFlag('V', (temp^uint16(c.A))&(temp^uint16(c.fetched))&0x0080 != 0)
	c.setFlag('N', temp&0x0080 != 0)
	c.A = byte(temp & 0x00FF)
}

func (c *CPU) adc() {
	c.fetch()
	temp := uint16(c.A) + uint16(c.fetched) + uint16(c.getFlag('C'))
	c.setFlag('C', temp > 255)
	c.setFlag('Z', (temp&0x00FF) == 0)
	c.setFlag('V', (^(uint16(c.A) ^ uint16(c.fetched)) & (uint16(c.A) ^ temp)) & 0x0080 != 0)
	c.setFlag('N', temp&0x80 != 0)
	c.A = byte(temp & 0x00FF)
}

func (c *CPU) dey() {
	c.Y--
	c.setFlag('Z', c.Y == 0)
	c.setFlag('N', c.Y&0x80 != 0)
}

func (c *CPU) dex() {
	c.X--
	c.setFlag('Z', c.X == 0)
	c.setFlag('N', c.X&0x80 != 0)
}

func (c *CPU) dec() {
	c.fetch()
	temp := c.fetched - 1
	c.bus.Write(c.addrAbs, temp)
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
}

func (c *CPU) iny() {
	c.Y++
	c.setFlag('Z', c.Y == 0)
	c.setFlag('N', c.Y&0x80 != 0)
}

func (c *CPU) inx() {
	c.X++
	c.setFlag('Z', c.X == 0)
	c.setFlag('N', c.X&0x80 != 0)
}

func (c *CPU) inc() {
	c.fetch()
	temp := c.fetched + 1
	c.bus.Write(c.addrAbs, temp)
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
}

func (c *CPU) eor() {
	c.fetch()
	c.A = c.A ^ c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
}

func (c *CPU) ora() {
	c.fetch()
	c.A = c.A | c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
}

func (c *CPU) and() {
	c.fetch()
	c.A = c.A & c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
}

func (c *CPU) ror() {
	c.fetch()
	temp := uint16(c.fetched) >> 1 | uint16(c.getFlag('C'))<<7
	c.setFlag('C', c.fetched&1 != 0)
	c.setFlag('Z', (temp&0x00FF) == 0)
	c.setFlag('N', temp&0x0080 != 0)
	if c.lookup[c.opcode].AddrModeName == "imp" {
		c.A = byte(temp & 0x00FF)
	} else {
		c.bus.Write(c.addrAbs, byte(temp&0x00FF))
	}
}

func (c *CPU) rol() {
	c.fetch()
	temp := uint16(c.fetched) << 1 | uint16(c.getFlag('C'))
	c.setFlag('C', temp > 0xFF)
	c.setFlag('Z', (temp&0x00FF) == 0)
	c.setFlag('N', temp&0x0080 != 0)
	if c.lookup[c.opcode].AddrModeName == "imp" {
		c.A = byte(temp & 0x00FF)
	} else {
		c.bus.Write(c.addrAbs, byte(temp&0x00FF))
	}
}

func (c *CPU) lsr() {
	c.fetch()
	c.setFlag('C', c.fetched&1 != 0)
	temp := c.fetched >> 1
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
	if c.lookup[c.opcode].AddrModeName == "imp" {
		c.A = temp
	} else {
		c.bus.Write(c.addrAbs, temp)
	}
}

func (c *CPU) asl() {
	c.fetch()
	temp := uint16(c.fetched) << 1
	c.setFlag('C', temp > 0xFF)
	c.setFlag('Z', (temp&0x00FF) == 0)
	c.setFlag('N', temp&0x0080 != 0)
	if c.lookup[c.opcode].AddrModeName == "imp" {
		c.A = byte(temp & 0x00FF)
	} else {
		c.bus.Write(c.addrAbs, byte(temp&0x00FF))
	}
}

func (c *CPU) branch() {
	c.cycles++
	c.addrAbs = c.PC + c.addrRel
	if (c.addrAbs & 0xFF00) != (c.PC & 0xFF00) {
		c.cycles++
	}
	c.PC = c.addrAbs
}

func (c *CPU) bvs() {
	if c.getFlag('V') == 1 {
		c.branch()
	}
}

func (c *CPU) bvc() {
	if c.getFlag('V') == 0 {
		c.branch()
	}
}

func (c *CPU) bpl() {
	if c.getFlag('N') == 0 {
		c.branch()
	}
}

func (c *CPU) bne() {
	if c.getFlag('Z') == 0 {
		c.branch()
	}
}

func (c *CPU) bmi() {
	if c.getFlag('N') == 1 {
		c.branch()
	}
}

func (c *CPU) beq() {
	if c.getFlag('Z') == 1 {
		c.branch()
	}
}

func (c *CPU) bcs() {
	if c.getFlag('C') == 1 {
		c.branch()
	}
}

func (c *CPU) bcc() {
	if c.getFlag('C') == 0 {
		c.branch()
	}
}

func (c *CPU) sei() {
	c.setFlag('I', true)
}

func (c *CPU) sed() {
	c.setFlag('D', true)
}

func (c *CPU) sec() {
	c.setFlag('C', true)
}

func (c *CPU) clv() {
	c.setFlag('V', false)
}

func (c *CPU) cli() {
	c.setFlag('I', false)
}

func (c *CPU) cld() {
	c.setFlag('D', false)
}

func (c *CPU) clc() {
	c.setFlag('C', false)
}

func (c *CPU) cpy() {
	c.fetch()
	temp := c.Y - c.fetched
	c.setFlag('C', c.Y >= c.fetched)
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
}

func (c *CPU) cpx() {
	c.fetch()
	temp := c.X - c.fetched
	c.setFlag('C', c.X >= c.fetched)
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
}

func (c *CPU) cmp() {
	c.fetch()
	temp := c.A - c.fetched
	c.setFlag('C', c.A >= c.fetched)
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
}

func (c *CPU) rti() {
	c.P = c.pop()
	c.setFlag(B, false)
	c.setFlag(U, false)

	c.PC = uint16(c.pop())
	c.PC |= uint16(c.pop()) << 8
}

func (c *CPU) rts() {
	c.PC = uint16(c.pop())
	c.PC |= uint16(c.pop()) << 8
	c.PC++
}

func (c *CPU) jsr() {
	c.PC--
	c.push(byte((c.PC >> 8) & 0x00FF))
	c.push(byte(c.PC & 0x00FF))
	c.PC = c.addrAbs
}

func (c *CPU) jmp() {
	c.PC = c.addrAbs
}

func (c *CPU) nop() {
	// Do nothing
}

func (c *CPU) bit() {
	c.fetch()
	temp := c.A & c.fetched
	c.setFlag('Z', temp == 0)
	c.setFlag('N', c.fetched&(1<<7) != 0)
	c.setFlag('V', c.fetched&(1<<6) != 0)
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
