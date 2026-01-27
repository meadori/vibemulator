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

	// --- Handle initial JMP $C5F5 from the provided nestest.nes ROM ---
	// The golden log expects execution to start as if an LDX #$00 was at C000.
	// Our ROM has JMP $C5F5 at C000. We execute this once and then align our cycle counts.

	// First instruction at C000 is JMP $C5F5 (opcode 0x4C)
	opcodeInitialJMP := mockBus.Read(c.PC)
	instrInitialJMP := c.Lookup[opcodeInitialJMP]

	// Ensure it's actually a JMP instruction (opcode 0x4C)
	if opcodeInitialJMP != 0x4C {
		log.Fatalf("Expected JMP $C5F5 (opcode 0x4C) at 0xC000, but found 0x%02X. Cannot align nestest.", opcodeInitialJMP)
	}

	// Execute the initial JMP instruction. This should consume 3 CPU cycles.
	cyclesConsumedByInitialJMP := instrInitialJMP.Cycles
	for j := 0; j < cyclesConsumedByInitialJMP; j++ {
		c.Clock()
	}
	// At this point, c.PC should be $C5F5, and 3 CPU cycles (9 PPU cycles) have passed.

	// Now, initialize totalCycles and totalPpuCycles to align with the golden log's
	// state *after* the JMP would have completed and *before* the first LDX #$00 is logged.
	// The golden log shows the first LDX #$00 at C000 (which we interpret as C5F5 in our ROM)
	// with PPU:  0, 27 and CYC:9.
	totalCycles := 9    // CYC:9 after JMP, before LDX
	totalPpuCycles := 27 // PPU:  0, 27 after JMP, before LDX

	// Loop and execute instructions, logging state
	for i := 0; i < 100000; i++ {
        // --- PREPARE FOR LOGGING ---
        // Capture CPU state *before* executing the current instruction
        pcToLog := c.PC
        aToLog := c.A
        xToLog := c.X
        yToLog := c.Y
        pToLog := c.P
        spToLog := c.SP

        // Read opcode and instruction details for logging
        opcodeToLog := mockBus.Read(pcToLog)
        instrToLog := c.Lookup[opcodeToLog]
        
        // Determine cycles this instruction will take
        cyclesThisInstruction := instrToLog.Cycles
        if instrToLog.Operate == nil {
            cyclesThisInstruction = 2 // Default for unimplemented
        }

        // --- EXECUTE THE INSTRUCTION ---
        for j := 0; j < cyclesThisInstruction; j++ {
            c.Clock()
        }
        // At this point, c.PC has advanced to the *next* instruction's address.
        // The current instruction has completed, and its cycles consumed.

        // --- UPDATE TOTAL CYCLES ---
        // The totalCycles should represent the cycles *after* this instruction has been processed.
        // The golden log has CYC as the cycle count *before* the instruction shown on the line starts.
        // So, the totalCycles for the next iteration needs to reflect this.
        totalCycles += cyclesThisInstruction
        totalPpuCycles += cyclesThisInstruction * 3

        // --- FORMAT AND PRINT LOG LINE ---
        ppuScanline := (totalPpuCycles - (cyclesThisInstruction * 3)) / 341 // PPU Scanline at start of instruction
        ppuPixel := (totalPpuCycles - (cyclesThisInstruction * 3)) % 341 // PPU Pixel at start of instruction
        
        // Construct the instruction and operand part of the log line for formatting
        opCodeAndOperandsRawPadded := ""
        operand1 := byte(0)
        operand2 := byte(0)
        switch instrToLog.AddrModeName {
        case "imm", "zp0", "zpx", "zpy", "rel", "izx", "izy":
            operand1 = mockBus.Read(pcToLog + 1)
            opCodeAndOperandsRawPadded = fmt.Sprintf("%02X %02X", opcodeToLog, operand1)
        case "abs", "abx", "aby", "ind", "jsr":
            operand1 = mockBus.Read(pcToLog + 1)
            operand2 = mockBus.Read(pcToLog + 2)
            opCodeAndOperandsRawPadded = fmt.Sprintf("%02X %02X %02X", opcodeToLog, operand1, operand2)
        case "imp":
            opCodeAndOperandsRawPadded = fmt.Sprintf("%02X", opcodeToLog)
        default:
            opCodeAndOperandsRawPadded = fmt.Sprintf("%02X", opcodeToLog) // Default for unknown addressing modes
        }

        // Construct the instruction disassembly string
        instructionDisassembly := ""
        switch instrToLog.AddrModeName {
        case "imp": instructionDisassembly = fmt.Sprintf("%s", instrToLog.Name)
        case "imm": instructionDisassembly = fmt.Sprintf("%s #$%02X", instrToLog.Name, operand1)
        case "zp0": instructionDisassembly = fmt.Sprintf("%s $%02X", instrToLog.Name, operand1)
        case "zpx": instructionDisassembly = fmt.Sprintf("%s $%02X,X", instrToLog.Name, operand1)
        case "zpy": instructionDisassembly = fmt.Sprintf("%s $%02X,Y", instrToLog.Name, operand1)
        case "rel":
            targetAddr := (pcToLog + uint16(2) + uint16(int8(operand1))) & 0xFFFF
            instructionDisassembly = fmt.Sprintf("%s $%04X", instrToLog.Name, targetAddr)
        case "abs": instructionDisassembly = fmt.Sprintf("%s $%04X", instrToLog.Name, (uint16(operand2)<<8)|uint16(operand1))
        case "abx": instructionDisassembly = fmt.Sprintf("%s $%04X,X", instrToLog.Name, (uint16(operand2)<<8)|uint16(operand1))
        case "aby": instructionDisassembly = fmt.Sprintf("%s $%04X,Y", instrToLog.Name, (uint16(operand2)<<8)|uint16(operand1))
        case "ind": instructionDisassembly = fmt.Sprintf("%s ($%04X)", instrToLog.Name, (uint16(operand2)<<8)|uint16(operand1))
        case "izx": instructionDisassembly = fmt.Sprintf("%s ($%02X,X)", instrToLog.Name, operand1)
        case "izy": instructionDisassembly = fmt.Sprintf("%s ($%02X),Y", instrToLog.Name, operand1)
        default: instructionDisassembly = fmt.Sprintf("%s ???", instrToLog.Name)
        }

        // Final log line construction
        logLine := fmt.Sprintf("%04X  %-8s %-32s A:%02X X:%02X Y:%02X P:%02X SP:%02X PPU:%3d,%3d CYC:%d",
            pcToLog,
            opCodeAndOperandsRawPadded,
            instructionDisassembly,
            aToLog, xToLog, yToLog, pToLog, spToLog,
            ppuScanline,
            ppuPixel,
            totalCycles - cyclesThisInstruction, // This is the change! CYC should be BEFORE the current instruction's cycles
        )

        fmt.Println(logLine)

		// Break conditions for nestest (usually a specific instruction or PC)
		// nestest will typically loop at a specific address (e.g., 0xC66A or 0xE000-ish) when finished.
		if c.PC == 0xC669 { // Example of a known end point for a nestest run.
			break
		}
	}
}
