package main

import (
	"fmt"
	"log"

	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/cpu"
)

// mockBus is a simple RAM implementation for the CPU to interact with.
type mockBus struct {
	Ram [65536]byte // Make Ram public
}

func (b *mockBus) Read(addr uint16) byte {
	return b.Ram[addr] // Use Ram
}

func (b *mockBus) Write(addr uint16, data byte) {
	b.Ram[addr] = data // Use Ram
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

	// Set Reset vector for nestest
	// nestest expects PC to be 0xC000 after reset, so set reset vector to 0xC000
	mockBus.Ram[0xFFFC] = 0x00
	mockBus.Ram[0xFFFD] = 0xC0

	// Load PRG ROM into mockBus
	// nestest ROM is typically 16KB PRG ROM (Mirrored)
	copy(mockBus.Ram[0x8000:], cart.PRGROM)
	if len(cart.PRGROM) == 16384 { // If 16KB PRG ROM, mirror it
		copy(mockBus.Ram[0xC000:], cart.PRGROM)
	} else { // Otherwise assume 32KB
		copy(mockBus.Ram[0xC000:], cart.PRGROM[16384:])
	}
	

	// Reset CPU
	c.Reset()
	// c.Reset() will set c.PC based on the reset vector (0xC000) and set c.Cycles = 8.
	// nestest requires initial SP to be 0xFD
	c.SP = 0xFD

	// Clock away initial reset cycles (8 cycles for 6502)
	for c.Cycles > 0 {
        c.Clock()
    }

	// Loop and execute instructions, logging state
	for i := 0; i < 100000; i++ { // Limit to 100,000 instructions for now
		fmt.Println(c.LogState())
		
		// The Clock() method itself fetches instructions when c.Cycles becomes 0.
		// So we just need to keep calling Clock until a new instruction is fetched
		// and its cycles consumed.
		
		// If c.Cycles is 0, the next Clock call will fetch an instruction and set c.Cycles.
		// If c.Cycles is > 0 (from previous instruction), we just clock it down.
		if c.Cycles == 0 {
			c.Clock() // Fetch first byte of new instruction, set c.Cycles
		}
		for c.Cycles > 0 { // Clock all cycles for the current instruction
			c.Clock()
		}

		// Break conditions for nestest (usually a specific instruction or PC)
		// nestest will typically loop at a specific address (e.g., 0xC66A or 0xE000-ish) when finished.
		if c.PC == 0xC669 { // Example of a known end point for a nestest run.
			break
		}
		if c.PC == 0xE000 { // Another common end point.
			break
		}
	}
}
