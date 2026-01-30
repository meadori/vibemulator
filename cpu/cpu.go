package cpu

import (
	"fmt"
)

// Declare logDebug function from main package
var LogDebug func(format string, a ...interface{})

// safeLogDebug calls LogDebug if it's not nil
func safeLogDebug(format string, a ...interface{}) {
	if LogDebug != nil {
		LogDebug(format, a...)
	}
}

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
	safeLogDebug("CPU Reset: PC = %04X", c.PC)

	c.A = 0
	c.X = 0
	c.Y = 0
	c.SP = 0xFD
	c.P = 0x00 | U
	c.setFlag('I', true) // This sets the I flag
	c.Cycles = 8 // Updated
}

// NMI is a non-maskable interrupt.
func (c *CPU) NMI() {
	c.push(byte((c.PC >> 8) & 0x00FF))
	c.push(byte(c.PC & 0x00FF))

	c.setFlag('B', false)
	c.setFlag('U', true)
	c.setFlag('I', true)
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
	safeLogDebug("CPU Clock")
	if c.Cycles == 0 {
		c.opcode = c.bus.Read(c.PC)
		c.PC++
		safeLogDebug("CPU Clock: PC = %04X, Opcode = %02X", c.PC, c.opcode)

		instr := c.Lookup[c.opcode]
		c.Cycles = instr.Cycles
		addedCycle1 := instr.AddrMode()
		addedCycle2 := instr.Operate()
		c.Cycles += int(addedCycle1 + addedCycle2)

	}
	c.Cycles--
}

func (c *CPU) push(data byte) {
	c.bus.Write(0x0100+uint16(c.SP), data)
	c.SP--
}

func (c *CPU) pop() byte {
	c.SP++
	return c.bus.Read(0x0100 + uint16(c.SP))
}


// createLookupTable creates and returns the 6502 instruction lookup table.
func (c *CPU) createLookupTable() [256]Instruction {
	lookup := [256]Instruction{
		// LDA
		0xA9: {"LDA", c.lda, c.imm, "imm", 2},
		0xA5: {"LDA", c.lda, c.zp0, "zp0", 3},
		0xB5: {"LDA", c.lda, c.zpx, "zpx", 4},
		0xAD: {"LDA", c.lda, c.abs, "abs", 4},
		0xBD: {"LDA", c.lda, c.abx, "abx", 4},
		0xB9: {"LDA", c.lda, c.aby, "aby", 4},
		0xA1: {"LDA", c.lda, c.izx, "izx", 6},
		0xB1: {"LDA", c.lda, c.izy, "izy", 5},

		// Unofficial Load (LAS)
		0xBB: {"LAS", c.las, c.aby, "aby", 4}, // LAS (LAR)

		// Unofficial Load (LAX)
		0xA7: {"LAX", c.lax, c.zp0, "zp0", 3},
		0xB7: {"LAX", c.lax, c.zpy, "zpy", 4},
		0xAF: {"LAX", c.lax, c.abs, "abs", 4},
		0xBF: {"LAX", c.lax, c.aby, "aby", 4},
		0xA3: {"LAX", c.lax, c.izx, "izx", 6},
				0xB3: {"LAX", c.lax, c.izy, "izy", 5},
		
				// Unofficial Load (LXA)
				0xAB: {"LXA", c.nop, c.imm, "imm", 2}, // LXA (LAX immediate) - Unstable, treat as NOP
		
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

		// Unofficial Store (SAX)
		0x87: {"SAX", c.sax, c.zp0, "zp0", 3},
		0x97: {"SAX", c.sax, c.zpy, "zpy", 4}, // zpy for SAX, not zpx
		0x8F: {"SAX", c.sax, c.abs, "abs", 4},
		0x83: {"SAX", c.sax, c.izx, "izx", 6},

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

		// Unofficial Increment/Decrement (DCP)
		0xC7: {"DCP", c.dcp, c.zp0, "zp0", 5},
		0xD7: {"DCP", c.dcp, c.zpx, "zpx", 6},
		0xCF: {"DCP", c.dcp, c.abs, "abs", 6},
		0xDF: {"DCP", c.dcp, c.abx, "abx", 7},
		0xDB: {"DCP", c.dcp, c.aby, "aby", 7},
		0xC3: {"DCP", c.dcp, c.izx, "izx", 8},
		0xD3: {"DCP", c.dcp, c.izy, "izy", 8},

		// Unofficial Arithmetic (ISC)
		0xE7: {"ISC", c.isc, c.zp0, "zp0", 5},
		0xF7: {"ISC", c.isc, c.zpx, "zpx", 6},
		0xEF: {"ISC", c.isc, c.abs, "abs", 6},
		0xFF: {"ISC", c.isc, c.abx, "abx", 7},
		0xFB: {"ISC", c.isc, c.aby, "aby", 7},
		0xE3: {"ISC", c.isc, c.izx, "izx", 8},
		0xF3: {"ISC", c.isc, c.izy, "izy", 8},


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

		// Unofficial Logical
		0x0B: {"ANC", c.anc, c.imm, "imm", 2}, // ANC
		0x2B: {"ANC", c.anc, c.imm, "imm", 2}, // ANC2
		0x4B: {"ALR", c.alr, c.imm, "imm", 2}, // ALR (ASR)
		0x8B: {"ANE", c.nop, c.imm, "imm", 2}, // ANE (XAA) - Unstable, treat as NOP
		0x6B: {"ARR", c.arr, c.imm, "imm", 2}, // ARR

		// Unofficial Shift/Rotate (RLA)
		0x27: {"RLA", c.rla, c.zp0, "zp0", 5},
		0x37: {"RLA", c.rla, c.zpx, "zpx", 6},
		0x2F: {"RLA", c.rla, c.abs, "abs", 6},
		0x3F: {"RLA", c.rla, c.abx, "abx", 7},
		0x3B: {"RLA", c.rla, c.aby, "aby", 7},
		0x23: {"RLA", c.rla, c.izx, "izx", 8},
		0x33: {"RLA", c.rla, c.izy, "izy", 8},

		// Unofficial Shift/Rotate (RRA)
		0x67: {"RRA", c.rra, c.zp0, "zp0", 5},
		0x77: {"RRA", c.rra, c.zpx, "zpx", 6},
		0x6F: {"RRA", c.rra, c.abs, "abs", 6},
		0x7F: {"RRA", c.rra, c.abx, "abx", 7},
		0x7B: {"RRA", c.rra, c.aby, "aby", 7},
		0x63: {"RRA", c.rra, c.izx, "izx", 8},
		0x73: {"RRA", c.rra, c.izy, "izy", 8},



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

	for i := 0; i < 256; i++ {
		if lookup[i].Operate == nil {
			lookup[i] = Instruction{"XXX", c.nop, c.imp, "imp", 2}
		}
	}
	return lookup
}


// Addressing Modes

func (c *CPU) imp() byte {
	c.fetched = c.A
	return 0
}

func (c *CPU) imm() byte {
	c.addrAbs = c.PC
	c.PC++
	return 0
}

func (c *CPU) zp0() byte {
	c.addrAbs = uint16(c.bus.Read(c.PC))
	c.PC++
	return 0
}

func (c *CPU) zpx() byte {
	c.addrAbs = uint16(c.bus.Read(c.PC) + c.X)
	c.PC++
	c.addrAbs &= 0x00FF
	return 0
}

func (c *CPU) zpy() byte {
	c.addrAbs = uint16(c.bus.Read(c.PC) + c.Y)
	c.PC++
	c.addrAbs &= 0x00FF
	return 0
}

func (c *CPU) rel() byte {
	c.addrRel = uint16(c.bus.Read(c.PC))
	c.PC++
	if c.addrRel&0x80 != 0 {
		c.addrRel |= 0xFF00
	}
	return 0
}

func (c *CPU) abs() byte {
	lo := uint16(c.bus.Read(c.PC))
	c.PC++
	hi := uint16(c.bus.Read(c.PC))
	c.PC++
	c.addrAbs = (hi << 8) | lo
	return 0
}

func (c *CPU) abx() byte {
	lo := uint16(c.bus.Read(c.PC))
	c.PC++
	hi := uint16(c.bus.Read(c.PC))
	c.PC++
	c.addrAbs = (hi << 8) | lo
	c.addrAbs += uint16(c.X)

	if (c.addrAbs & 0xFF00) != (hi << 8) {
		return 1
	}
	return 0
}

func (c *CPU) aby() byte {
	lo := uint16(c.bus.Read(c.PC))
	c.PC++
	hi := uint16(c.bus.Read(c.PC))
	c.PC++
	c.addrAbs = (hi << 8) | lo
	c.addrAbs += uint16(c.Y)

	if (c.addrAbs & 0xFF00) != (hi << 8) {
		return 1
	}
	return 0
}

func (c *CPU) ind() byte {
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
	return 0
}

func (c *CPU) izx() byte {
	t := uint16(c.bus.Read(c.PC))
	c.PC++
	lo := uint16(c.bus.Read((t + uint16(c.X)) & 0x00FF))
	hi := uint16(c.bus.Read((t + uint16(c.X) + 1) & 0x00FF))
	c.addrAbs = (hi << 8) | lo
	return 0
}

func (c *CPU) izy() byte {
	t := uint16(c.bus.Read(c.PC))
	c.PC++
	lo := uint16(c.bus.Read(t & 0x00FF))
	hi := uint16(c.bus.Read((t + 1) & 0x00FF))
	c.addrAbs = (hi << 8) | lo
	c.addrAbs += uint16(c.Y)

	if (c.addrAbs & 0xFF00) != (hi << 8) {
		return 1
	}
	return 0
}

// Instructions

func (c *CPU) ldy() byte {
	c.fetch()
	c.Y = c.fetched
	c.setFlag('Z', c.Y == 0)
	c.setFlag('N', c.Y&0x80 != 0)
	return 0
}

func (c *CPU) ldx() byte {
	c.fetch()
	c.X = c.fetched
	c.setFlag('Z', c.X == 0)
	c.setFlag('N', c.X&0x80 != 0)
	return 0
}

func (c *CPU) sty() byte {
	c.bus.Write(c.addrAbs, c.Y)
	return 0
}

func (c *CPU) stx() byte {
	c.bus.Write(c.addrAbs, c.X)
	return 0
}

func (c *CPU) sta() byte {
	c.bus.Write(c.addrAbs, c.A)
	return 0
}

func (c *CPU) sax() byte {
	val := c.A & c.X
	c.bus.Write(c.addrAbs, val)
	return 0
}

func (c *CPU) plp() byte {
	c.P = c.pop()
	c.setFlag(B, false) // Explicitly clear B flag (bit 4)
	c.setFlag(U, true)  // Explicitly set U flag (bit 5)
	return 0
}

func (c *CPU) php() byte {
	c.push(c.P | B | U)
	return 0
}

func (c *CPU) pla() byte {
	c.A = c.pop()
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	return 0
}

func (c *CPU) pha() byte {
	c.push(c.A)
	return 0
}

func (c *CPU) tya() byte {
	c.A = c.Y
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	return 0
}

func (c *CPU) tay() byte {
	c.Y = c.A
	c.setFlag('Z', c.Y == 0)
	c.setFlag('N', c.Y&0x80 != 0)
	return 0
}

func (c *CPU) txa() byte {
	c.A = c.X
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	return 0
}

func (c *CPU) tsx() byte {
	c.X = c.SP
	c.setFlag('Z', c.X == 0)
	c.setFlag('N', c.X&0x80 != 0)
	return 0
}

func (c *CPU) txs() byte {
	c.SP = c.X
	return 0
}

func (c *CPU) tax() byte {
	c.X = c.A
	c.setFlag('Z', c.X == 0)
	c.setFlag('N', c.X&0x80 != 0)
	return 0
}

func (c *CPU) lda() byte {
	c.fetch()
	c.A = c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	return 0
}

func (c *CPU) las() byte {
	c.fetch()
	val := c.fetched & c.SP
	c.A = val
	c.X = val
	c.SP = val
	c.setFlag('Z', val == 0)
	c.setFlag('N', val&0x80 != 0)
	return 0
}

func (c *CPU) lax() byte {
	c.fetch()
	c.A = c.fetched
	c.X = c.A // TAX operation
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	return 0
}

func (c *CPU) sbc() byte {
	c.fetch()
	temp := uint16(c.A) - uint16(c.fetched) - (1 - uint16(c.getFlag('C')))
	c.setFlag('C', temp < 0x100)
	c.setFlag('Z', (temp&0x00FF) == 0)
	c.setFlag('V', ((uint16(c.A) ^ temp) & (0x00FF ^ uint16(c.fetched) ^ temp)) & 0x0080 != 0)
	c.setFlag('N', temp&0x0080 != 0)
	c.A = byte(temp & 0x00FF)
	return 1
}

func (c *CPU) adc() byte {
	c.fetch()
	temp := uint16(c.A) + uint16(c.fetched) + uint16(c.getFlag('C'))
	c.setFlag('C', temp > 255)
	c.setFlag('Z', (temp&0x00FF) == 0)
	c.setFlag('V', ((uint16(c.A) ^ temp) & (uint16(c.fetched) ^ temp)) & 0x0080 != 0)
	c.setFlag('N', temp&0x80 != 0)
	c.A = byte(temp & 0x00FF)
	return 1
}

func (c *CPU) dey() byte {
	c.Y--
	c.setFlag('Z', c.Y == 0)
	c.setFlag('N', c.Y&0x80 != 0)
	return 0
}

func (c *CPU) dex() byte {
	c.X--
	c.setFlag('Z', c.X == 0)
	c.setFlag('N', c.X&0x80 != 0)
	return 0
}

func (c *CPU) dec() byte {
	c.fetch()
	temp := c.fetched - 1
	c.bus.Write(c.addrAbs, temp)
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
	return 0
}

func (c *CPU) iny() byte {
	c.Y++
	c.setFlag('Z', c.Y == 0)
	c.setFlag('N', c.Y&0x80 != 0)
	return 0
}

func (c *CPU) inx() byte {
	c.X++
	c.setFlag('Z', c.X == 0)
	c.setFlag('N', c.X&0x80 != 0)
	return 0
}

func (c *CPU) inc() byte {
	c.fetch()
	temp := c.fetched + 1
	c.bus.Write(c.addrAbs, temp)
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
	return 0
}

func (c *CPU) dcp() byte {
	c.fetch()
	// DEC operation
	temp := c.fetched - 1
	c.bus.Write(c.addrAbs, temp)

	// CMP operation
	res := c.A - temp
	c.setFlag('C', c.A >= temp)
	c.setFlag('Z', res == 0)
	c.setFlag('N', res&0x80 != 0)
	return 0
}

func (c *CPU) isc() byte {
	c.fetch()
	// INC operation
	temp := c.fetched + 1
	c.bus.Write(c.addrAbs, temp)

	// SBC operation (similar to regular SBC, but with the incremented value)
	sbcVal := uint16(temp)
	res := uint16(c.A) - sbcVal - (1 - uint16(c.getFlag('C')))

	c.setFlag('C', res < 0x100) // If borrow, C is clear
	c.setFlag('Z', (res&0x00FF) == 0)
	c.setFlag('V', ((uint16(c.A) ^ res) & (sbcVal ^ res)) & 0x0080 != 0)
	c.setFlag('N', res&0x0080 != 0)
	c.A = byte(res & 0x00FF)
	return 0
}

func (c *CPU) eor() byte {
	c.fetch()
	c.A = c.A ^ c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	return 0
}

func (c *CPU) anc() byte {
	c.fetch()
	c.A = c.A & c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	c.setFlag('C', c.getFlag('N') == 1) // Set Carry flag to the value of the Negative flag
	return 0
}

func (c *CPU) and() byte {
	c.fetch()
	c.A = c.A & c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	return 0
}

func (c *CPU) ora() byte {
	c.fetch()
	c.A = c.A | c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	return 0
}

func (c *CPU) alr() byte {
	c.fetch()
	c.A = c.A & c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)

	c.setFlag('C', c.A&1 != 0)
	c.A = c.A >> 1
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	return 0
}

func (c *CPU) ror() byte {
	c.fetch()
	temp := uint16(c.fetched) >> 1 | uint16(c.getFlag('C'))<<7
	c.setFlag('C', c.fetched&1 != 0)
	c.setFlag('Z', (temp&0x00FF) == 0)
	c.setFlag('N', temp&0x0080 != 0)
	if c.Lookup[c.opcode].AddrModeName == "imp" {
		c.A = byte(temp & 0x00FF)
	} else {
		c.bus.Write(c.addrAbs, byte(temp&0x00FF))
	}
	return 0
}

func (c *CPU) arr() byte {
	c.fetch()
	c.A = c.A & c.fetched
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)

	// ROR operation
	oldC := c.getFlag('C')
	c.setFlag('C', c.A&1 != 0)
	c.A = (c.A >> 1) | (oldC << 7)

	// Update N, Z flags based on new A
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)

	// ARR specific V flag update
	c.setFlag('V', ((c.A>>6)&1)^((c.A>>5)&1) != 0)

	return 0
}

func (c *CPU) rol() byte {
	c.fetch()
	temp := uint16(c.fetched) << 1 | uint16(c.getFlag('C'))
	c.setFlag('C', temp > 0xFF)
	c.setFlag('Z', (temp&0x00FF) == 0)
	c.setFlag('N', temp&0x0080 != 0)
	if c.Lookup[c.opcode].AddrModeName == "imp" {
		c.A = byte(temp & 0x00FF)
	} else {
		c.bus.Write(c.addrAbs, byte(temp&0x00FF))
	}
	return 0
}

func (c *CPU) lsr() byte {
	c.fetch()
	c.setFlag('C', c.fetched&1 != 0)
	temp := c.fetched >> 1
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
	if c.Lookup[c.opcode].AddrModeName == "imp" {
		c.A = temp
	} else {
		c.bus.Write(c.addrAbs, temp)
	}
	return 0
}

func (c *CPU) asl() byte {
	c.fetch()
	temp := uint16(c.fetched) << 1
	c.setFlag('C', temp > 0xFF)
	c.setFlag('Z', (temp&0x00FF) == 0)
	c.setFlag('N', temp&0x0080 != 0)
	if c.Lookup[c.opcode].AddrModeName == "imp" {
		c.A = byte(temp & 0x00FF)
	} else {
		c.bus.Write(c.addrAbs, byte(temp&0x00FF))
	}
	return 0
}

func (c *CPU) rla() byte {
	c.fetch()
	val := c.fetched

	// ROL operation
	oldC := c.getFlag('C')
	c.setFlag('C', val&0x80 != 0)
	val = (val << 1) | oldC

	c.bus.Write(c.addrAbs, val) // Write back rotated value

	// AND operation
	c.A = c.A & val
	c.setFlag('Z', c.A == 0)
	c.setFlag('N', c.A&0x80 != 0)
	return 0
}

func (c *CPU) rra() byte {
	c.fetch()
	val := c.fetched

	// ROR operation
	oldC := c.getFlag('C')
	c.setFlag('C', val&1 != 0)
	val = (val >> 1) | (oldC << 7)

	c.bus.Write(c.addrAbs, val) // Write back rotated value

	// ADC operation (similar to regular ADC, but with the rotated value)
	adcVal := uint16(val)
	res := uint16(c.A) + adcVal + uint16(c.getFlag('C'))

	c.setFlag('C', res > 255)
	c.setFlag('Z', (res&0x00FF) == 0)
	c.setFlag('V', ((uint16(c.A) ^ res) & (adcVal ^ res)) & 0x0080 != 0)
	c.setFlag('N', res&0x80 != 0)
	c.A = byte(res & 0x00FF)
	return 0
}

func (c *CPU) bvs() byte {
	if c.getFlag('V') == 1 {
		c.branch()
	}
	return 0
}

func (c *CPU) bvc() byte {
	if c.getFlag('V') == 0 {
		c.branch()
	}
	return 0
}

func (c *CPU) bpl() byte {
	if c.getFlag('N') == 0 {
		c.branch()
	}
	return 0
}

func (c *CPU) bne() byte {
	if c.getFlag('Z') == 0 {
		c.branch()
	}
	return 0
}

func (c *CPU) bmi() byte {
	if c.getFlag('N') == 1 {
		c.branch()
	}
	return 0
}

func (c *CPU) beq() byte {
	if c.getFlag('Z') == 1 {
		c.branch()
	}
	return 0
}

func (c *CPU) bcs() byte {
	if c.getFlag('C') == 1 {
		c.branch()
	}
	return 0
}

func (c *CPU) bcc() byte {
	if c.getFlag('C') == 0 {
		c.branch()
	}
	return 0
}

func (c *CPU) sei() byte {
	c.setFlag('I', true)
	return 0
}

func (c *CPU) sed() byte {
	c.setFlag('D', true)
	return 0
}

func (c *CPU) sec() byte {
	c.setFlag('C', true)
	return 0
}

func (c *CPU) clv() byte {
	c.setFlag('V', false)
	return 0
}

func (c *CPU) cli() byte {
	c.setFlag('I', false)
	return 0
}

func (c *CPU) cld() byte {
	c.setFlag('D', false)
	return 0
}

func (c *CPU) clc() byte {
	c.setFlag('C', false)
	return 0
}

func (c *CPU) cpy() byte {
	c.fetch()
	temp := c.Y - c.fetched
	c.setFlag('C', c.Y >= c.fetched)
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
	return 0
}

func (c *CPU) cpx() byte {
	c.fetch()
	temp := c.X - c.fetched
	c.setFlag('C', c.X >= c.fetched)
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
	return 0
}

func (c *CPU) cmp() byte {
	c.fetch()
	temp := c.A - c.fetched
	c.setFlag('C', c.A >= c.fetched)
	c.setFlag('Z', temp == 0)
	c.setFlag('N', temp&0x80 != 0)
	return 1
}

func (c *CPU) rti() byte {
	c.P = c.pop()
	c.setFlag(B, false)
	c.setFlag(U, false)

	c.PC = uint16(c.pop())
	c.PC |= uint16(c.pop()) << 8
	return 0
}

func (c *CPU) rts() byte {
	c.PC = uint16(c.pop())
	c.PC |= uint16(c.pop()) << 8
	c.PC++
	return 0
}

func (c *CPU) jsr() byte {
	c.PC--
	c.push(byte((c.PC >> 8) & 0x00FF))
	c.push(byte(c.PC & 0x00FF))
	c.PC = c.addrAbs
	return 0
}

func (c *CPU) jmp() byte {
	c.PC = c.addrAbs
	return 0
}

func (c *CPU) nop() byte {
	// Do nothing
	return 0
}

func (c *CPU) bit() byte {
	c.fetch()
	temp := c.A & c.fetched
	c.setFlag('Z', temp == 0)
	c.setFlag('N', c.fetched&(1<<7) != 0)
	c.setFlag('V', c.fetched&(1<<6) != 0)
	return 0
}

func (c *CPU) fetch() byte {
	if c.Lookup[c.opcode].AddrModeName != "imp" {
		c.fetched = c.bus.Read(c.addrAbs)
	}
	return 0
}

func (c *CPU) branch() byte {
	c.Cycles++
	c.addrAbs = c.PC + c.addrRel

	if (c.addrAbs & 0xFF00) != (c.PC & 0xFF00) {
		c.Cycles++
	}
	c.PC = c.addrAbs
	return 0
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
