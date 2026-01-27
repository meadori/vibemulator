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

	// nestest expects PC to be 0xC000 after reset, so we rely on the ROM's reset vector.

	// Load PRG ROM into mockBus for nestest
	// Assume nestest code is in the first 16KB of PRGROM, and mirror it.
	// This handles both 16KB and 32KB nestest ROMs consistently for the test.
	copy(mockBus.Ram[0x8000:], cart.PRGROM[:16384]) // Copy first 16KB to 0x8000
	copy(mockBus.Ram[0xC000:], cart.PRGROM[:16384]) // Mirror first 16KB to 0xC000

	// Reset CPU
	c.Reset()
	// c.Reset() will set c.PC based on the ROM's reset vector.
	// For nestest, we want to start execution at 0xC000 regardless.
	c.PC = 0xC000 // Explicitly set PC to 0xC000 for nestest

	// nestest requires initial SP to be 0xFD
	c.SP = 0xFD

	totalCycles := 0 // New: total cycle counter

    // Clock away initial reset cycles (8 cycles for 6502)
    // The c.Cycles are already managed by the Clock() method
    // After Reset, c.Cycles = 8.
    // Each Clock() call decrements c.Cycles.
    // The very first instruction fetch happens when c.Cycles becomes 0.
    for c.Cycles > 0 {
        c.Clock()
        totalCycles++ // Increment total cycles here too
    }

	// Loop and execute instructions, logging state
	for i := 0; i < 1; i++ { // Limit to 1 instruction for now
        // --- Capture state before instruction execution for logging ---
        pcBefore := c.PC
        aBefore := c.A
        xBefore := c.X
        yBefore := c.Y
        pBefore := c.P
        spBefore := c.SP

        // Determine opcode and operands for current instruction
        opcode := mockBus.Read(pcBefore)
        instr := c.Lookup[opcode]
        
        // --- Execute one full instruction ---
        if c.Cycles == 0 {
            c.Clock() // Fetch first byte of new instruction, sets c.Cycles for *this* instruction
        }
        cyclesThisInstruction := c.Cycles // c.Cycles will now hold the cycles needed for this instruction
        for j := 0; j < cyclesThisInstruction; j++ { // Clock all cycles for the current instruction
            c.Clock()
            totalCycles++ // Increment total cycles for each clock
        }


        // --- Format and Print log line ---
        // Example: C000  A2 00     LDX #$00                        A:00 X:00 Y:00 P:24 SP:FD PPU:  0,  0 CYC:0
        
        // PPU cycles are not emulated in mockBus context, so use placeholder "  0,  0"
        ppuCycleStr := "  0,  0" 

        // Construct the instruction and operand part of the log line for formatting
        opCodeAndOperandsRawPadded := ""
        operand1 := byte(0)
        operand2 := byte(0)
        switch instr.AddrModeName {
        case "imm", "zp0", "zpx", "zpy", "rel", "izx", "izy":
            operand1 = mockBus.Read(pcBefore + 1)
            opCodeAndOperandsRawPadded = fmt.Sprintf("%02X %02X", opcode, operand1)
        case "abs", "abx", "aby", "ind", "jsr":
            operand1 = mockBus.Read(pcBefore + 1)
            operand2 = mockBus.Read(pcBefore + 2)
            opCodeAndOperandsRawPadded = fmt.Sprintf("%02X %02X %02X", opcode, operand1, operand2)
        case "imp": 
            opCodeAndOperandsRawPadded = fmt.Sprintf("%02X", opcode)
        default:
            opCodeAndOperandsRawPadded = fmt.Sprintf("%02X", opcode) // Default for unknown addressing modes
        }

        // Construct the instruction disassembly string
        instructionDisassembly := ""
        switch instr.AddrModeName {
        case "imp": instructionDisassembly = fmt.Sprintf("%s", instr.Name)
        case "imm": instructionDisassembly = fmt.Sprintf("%s #$%02X", instr.Name, operand1)
        case "zp0": instructionDisassembly = fmt.Sprintf("%s $%02X", instr.Name, operand1)
        case "zpx": instructionDisassembly = fmt.Sprintf("%s $%02X,X", instr.Name, operand1)
        case "zpy": instructionDisassembly = fmt.Sprintf("%s $%02X,Y", instr.Name, operand1)
        case "rel": 
            targetAddr := (pcBefore + uint16(2) + uint16(int8(operand1))) & 0xFFFF 
            instructionDisassembly = fmt.Sprintf("%s $%04X", instr.Name, targetAddr) 
        case "abs": instructionDisassembly = fmt.Sprintf("%s $%04X", instr.Name, (uint16(operand2)<<8)|uint16(operand1))
        case "abx": instructionDisassembly = fmt.Sprintf("%s $%04X,X", instr.Name, (uint16(operand2)<<8)|uint16(operand1))
        case "aby": instructionDisassembly = fmt.Sprintf("%s $%04X,Y", instr.Name, (uint16(operand2)<<8)|uint16(operand1))
        case "ind": instructionDisassembly = fmt.Sprintf("%s ($%04X)", instr.Name, (uint16(operand2)<<8)|uint16(operand1))
        case "izx": instructionDisassembly = fmt.Sprintf("%s ($%02X,X)", instr.Name, operand1)
        case "izy": instructionDisassembly = fmt.Sprintf("%s ($%02X),Y", instr.Name, operand1)
        default: instructionDisassembly = fmt.Sprintf("%s ???", instr.Name)
        }
        
        // Final log line construction
        logLine := fmt.Sprintf("%04X  %-8s %-32s A:%02X X:%02X Y:%02X P:%02X SP:%02X PPU:%s CYC:%d",
            pcBefore,
            opCodeAndOperandsRawPadded, 
            instructionDisassembly,
            aBefore, xBefore, yBefore, pBefore, spBefore,
            ppuCycleStr,
            totalCycles, // totalCycles is the cycles *after* this instruction
        )
        
        fmt.Println(logLine)

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
