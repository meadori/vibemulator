package main

import (
	"fmt"
	"log"
	"os"

	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/cpu"
)

// mockBus is a simple RAM implementation for the CPU to interact with.
type mockBus struct {
	ram [65536]byte
}

func (b *mockBus) Read(addr uint16) byte {
	return b.ram[addr]
}

func (b *mockBus) Write(addr uint16, data byte) {
	b.ram[addr] = data
}

// executeOneInstruction fully executes a single CPU instruction.
func executeOneInstruction(c *cpu.CPU, mockBus *mockBus) {
	// Clock any remaining cycles from previous operations (e.g., Reset)
	for c.Cycles > 0 { // Updated to c.Cycles
		c.Clock()
	}

	// Determine cycles for the next instruction *before* Clock() fetches it.
	// We need to read the opcode from the bus associated with the CPU's PC
	opcode := mockBus.Read(c.PC)
	instr := c.Lookup[opcode] // Use c.Lookup now that it's public
	cyclesToConsume := instr.Cycles

	// Clock the CPU until this instruction is fully executed.
	for i := 0; i < cyclesToConsume; i++ {
		c.Clock()
	}
}

func main() {
	romPath := "nestest/testdata/nestest.nes" // Hardcoded path

	cart, err := cartridge.New(romPath)
	if err != nil {
		log.Fatalf("Error loading nestest ROM from %s: %v. Please ensure a valid nestest.nes is placed there.", romPath, err)
	}

	c := cpu.New()
	mockBus := &mockBus{}
	c.ConnectBus(mockBus)

	// Load PRG ROM into mockBus
	// nestest ROM is typically 16KB PRG ROM (Mirrored)
	copy(mockBus.ram[0x8000:], cart.PRGROM)
	if len(cart.PRGROM) == 16384 { // If 16KB PRG ROM, mirror it
		copy(mockBus.ram[0xC000:], cart.PRGROM)
	} else { // Otherwise assume 32KB
		copy(mockBus.ram[0xC000:], cart.PRGROM[16384:])
	}
	

	// Reset CPU
	c.Reset()
	// After Reset, c.Cycles is 8. Clock these away so CPU is ready to fetch.
	for i := 0; i < 8; i++ { // Changed from c.cycles to c.Cycles (Clock() will use c.Cycles)
		c.Clock()
	}

	// For nestest.nes, execution typically starts at 0xC000.
	// The Reset vector in nestest usually points to 0xC000.
	// We explicitly set PC to 0xC000 to match nestest's expectation.
	// If the cart.PRGROM was 32K, then the 2nd bank would be at 0xC000
	// If the cart.PRGROM was 16K, then it would be mirrored to 0xC000
	c.PC = 0xC000
	
	// nestest requires initial SP to be 0xFD
	c.SP = 0xFD

	// Loop and execute instructions, logging state
	for i := 0; i < 100000; i++ { // Limit to 100,000 instructions for now
		fmt.Println(c.LogState())
		
		// Break conditions for nestest (usually a specific instruction or PC)
		// nestest will typically loop at a specific address (e.g., 0xC66A or 0xE000-ish) when finished.
		// For now, just a loop limit.
		if c.PC == 0xC669 { // This is an example from some nestest runners.
			break
		}
		if c.PC == 0xE000 { // Another common end point.
			break
		}

		executeOneInstruction(c, mockBus)
	}
}
