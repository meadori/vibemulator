package cpu

import (
	"testing"
)

type mockBus struct {
	ram [65536]byte
}

func (b *mockBus) Read(addr uint16) byte {
	return b.ram[addr]
}

func (b *mockBus) Write(addr uint16, data byte) {
	b.ram[addr] = data
}

func executeOneInstruction(c *CPU) {
	// First, clock any remaining cycles from previous operations (e.g., Reset)
	for c.cycles > 0 {
		c.Clock()
	}

	// Now c.cycles is 0, the next Clock() call will fetch and process the instruction.
	// We need to determine the total cycles this instruction will consume *after* it's fetched.
	// This requires peeking at the opcode at c.PC before Clock() consumes it.
	opcode := c.bus.Read(c.PC)
	instr := c.lookup[opcode]
	cyclesToConsume := instr.Cycles

	// Clock the CPU until this instruction is fully executed.
	// The first Clock() call when c.cycles == 0 will fetch and update c.cycles.
	// The subsequent calls will decrement c.cycles.
	for i := 0; i < cyclesToConsume; i++ {
		c.Clock()
	}
}

func setupCPU(t *testing.T) (*CPU, *mockBus) {
	c := New()
	bus := &mockBus{}
	c.ConnectBus(bus)
	c.Reset()
	// After Reset, c.cycles is 8. Clock these away so CPU is ready to fetch.
	for i := 0; i < 8; i++ {
		c.Clock()
	}
	c.PC = 0x8000
	return c, bus
}

func TestLoadStore(t *testing.T) {
	c, bus := setupCPU(t)

	// LDA IMM
	bus.Write(0x8000, 0xA9)
	bus.Write(0x8001, 0x42)
	executeOneInstruction(c) // Replaced c.Clock()
	if c.A != 0x42 {
		t.Error("LDA IMM failed")
	}

	// STA ABS
	c.PC = 0x8002
	bus.Write(0x8002, 0x8D)
	bus.Write(0x8003, 0x10)
	bus.Write(0x8004, 0x01)
	executeOneInstruction(c) // Replaced c.Clock()
	if bus.ram[0x0110] != 0x42 {
		t.Error("STA ABS failed")
	}
}

func TestArithmetic(t *testing.T) {
	c, bus := setupCPU(t)

	// ADC
	c.A = 10
	bus.Write(0x8000, 0x69) // ADC #$05
	bus.Write(0x8001, 5)
	executeOneInstruction(c) // Replaced c.Clock()
	if c.A != 15 {
		t.Error("ADC failed")
	}

	// SBC
	c.PC = 0x8002
	c.setFlag('C', true)
	bus.Write(0x8002, 0xE9) // SBC #$05
	bus.Write(0x8003, 5)
	executeOneInstruction(c) // Replaced c.Clock()
	if c.A != 10 {
		t.Error("SBC failed")
	}
}

func TestIncDec(t *testing.T) {
	c, bus := setupCPU(t)

	// INC
	bus.Write(0x10, 0x41)
	bus.Write(0x8000, 0xE6) // INC $10
	bus.Write(0x8001, 0x10)
	executeOneInstruction(c) // Replaced c.Clock()
	if bus.ram[0x10] != 0x42 {
		t.Error("INC failed")
	}

	// INX
	c.PC = 0x8002
	c.X = 0x10
	bus.Write(0x8002, 0xE8) // INX
	executeOneInstruction(c) // Replaced c.Clock()
	if c.X != 0x11 {
		t.Error("INX failed")
	}
}

func TestLogical(t *testing.T) {
	c, bus := setupCPU(t)

	// AND
	c.A = 0b10101010
	bus.Write(0x8000, 0x29) // AND #$0F
	bus.Write(0x8001, 0b00001111)
	executeOneInstruction(c) // Replaced c.Clock()
	if c.A != 0b00001010 {
		t.Error("AND failed")
	}
}

func TestShiftRotate(t *testing.T) {
	c, bus := setupCPU(t)

	// ASL
	c.A = 0b01010101
	bus.Write(0x8000, 0x0A) // ASL
	executeOneInstruction(c) // Replaced c.Clock()
	if c.A != 0b10101010 {
		t.Error("ASL failed")
	}
	if c.getFlag('C') != 0 {
		t.Error("ASL carry failed")
	}

	// LSR
	c.PC = 0x8001
	bus.Write(0x8001, 0x4A) // LSR
	executeOneInstruction(c) // Replaced c.Clock()
	if c.A != 0b01010101 {
		t.Error("LSR failed")
	}
	if c.getFlag('C') != 0 {
		t.Error("LSR carry failed")
	}
}

func TestBranch(t *testing.T) {
	c, bus := setupCPU(t)

	// BEQ (not taken)
	bus.Write(0x8000, 0xF0) // BEQ $10
	bus.Write(0x8001, 0x10)
	executeOneInstruction(c) // Replaced c.Clock()
	if c.PC != 0x8002 {
		t.Error("BEQ (not taken) failed")
	}

	// BEQ (taken)
	c.PC = 0x8002
	c.setFlag('Z', true)
	bus.Write(0x8002, 0xF0) // BEQ $10
	bus.Write(0x8003, 0x10)
	executeOneInstruction(c) // Replaced c.Clock()
	if c.PC != 0x8014 {
		t.Error("BEQ (taken) failed")
	}
}
