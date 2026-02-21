package ppu

import (
	"image/color"
	"testing"

	"github.com/meadori/vibemulator/cartridge"
)

// mockBus for CPU to interact with, similar to nestest/main.go
type mockBus struct {
	PPU *PPU
	Ram [65536]byte // 64KB of RAM for CPU address space
}

// mockMapper implements the cartridge.Mapper interface for testing purposes.
type mockMapper struct {
	chrROM    []byte
	mirroring byte
}

func (m *mockMapper) CPUMapRead(addr uint16) (byte, bool) {
	return 0, false // Not used in this PPU test
}

func (m *mockMapper) CPUMapWrite(addr uint16, data byte) bool {
	return false // Not used in this PPU test
}

func (m *mockMapper) PPUMapRead(addr uint16) (byte, bool) {
	if addr >= 0x0000 && addr <= 0x1FFF {
		return m.chrROM[addr], true
	}
	return 0, false
}

func (m *mockMapper) PPUMapWrite(addr uint16, data byte) bool {
	if addr >= 0x0000 && addr <= 0x1FFF {
		m.chrROM[addr] = data
		return true
	}
	return false
}

func (m *mockMapper) Clock() {}

func (m *mockMapper) IRQPending() bool { return false }
func (m *mockMapper) ClearIRQ()        {}

func (m *mockMapper) GetMirroring() byte {
	return m.mirroring
}

// createTestCartridge generates a minimal cartridge for PPU background testing.
// It sets up CHR-ROM with a specific pattern and PRG-ROM for CPU to configure PPU.
func createTestCartridge() *cartridge.Cartridge {
	// Create a minimal PRG-ROM (16KB) - just enough for some CPU init code
	prgROM := make([]byte, 0x4000) // 16KB

	// Minimal CPU code to initialize PPU for background rendering
	// We need to write to PPUCTRL ($2000) and PPUMASK ($2001)
	// PPUCTRL (0x2000): Set nametable address, enable NMI (not critical for render test)
	// PPUMASK (0x2001): Enable background rendering, enable sprites (optional)
	// Example: Enable background, enable sprites, no clipping
	// PPUCTRL: 0x90 (VBLANK NMI, BG at 0x2400)
	// PPUMASK: 0x1E (Enable BG, Enable Sprites, Show Left BG/Sprite)

	// CPU code to write PPUCTRL and PPUMASK
	// A9 NN    ; LDA #NN
	// 8D 00 20 ; STA $2000
	// A9 NN    ; LDA #NN
	// 8D 01 20 ; STA $2001
	// 4C XX YY ; JMP $YYXX (infinite loop)

	// Set PPUCTRL to enable NMI, background at $2000 (0x20)
	prgROM[0x0000] = 0xA9 // LDA #
	prgROM[0x0001] = 0x20 // Value for PPUCTRL (NMI enable, nametable $2000)
	prgROM[0x0002] = 0x8D // STA $XXXX
	prgROM[0x0003] = 0x00 // Low byte of $2000
	prgROM[0x0004] = 0x20 // High byte of $2000

	// Set PPUMASK to enable background rendering (0x08), enable sprites (0x10), no clipping (0x02, 0x04)
	prgROM[0x0005] = 0xA9 // LDA #
	prgROM[0x0006] = 0x1E // Value for PPUMASK (enable rendering, show left 8px)
	prgROM[0x0007] = 0x8D // STA $XXXX
	prgROM[0x0008] = 0x01 // Low byte of $2001
	prgROM[0x0009] = 0x20 // High byte of $2001

	// Infinite loop
	prgROM[0x000A] = 0x4C // JMP $XXXX
	prgROM[0x000B] = 0x0A // Low byte of $000A
	prgROM[0x000C] = 0x00 // High byte of $000A

	// Set reset vector for CPU to start at our code
	prgROM[0x3FFC] = 0x00 // Low byte of start address
	prgROM[0x3FFD] = 0x80 // High byte of start address ($8000)

	// CHR-ROM (8KB) - Pattern Table
	// We'll define a simple 8x8 tile: a solid red square.
	// Each tile is 16 bytes: 8 for LSB plane, 8 for MSB plane.
	// Color 0: transparent (or background color)
	// Color 1: red
	// Color 2: green
	// Color 3: blue

	// Tile 0: Solid Red (color 1)
	// LSB plane: All 1s
	// MSB plane: All 0s
	// This makes pixel color = (MSB << 1) | LSB
	// So (0 << 1) | 1 = 1 (red)
	chrROM := make([]byte, 0x2000) // 8KB

	// Tile 0 LSB plane (all 1s)
	for i := 0; i < 8; i++ {
		chrROM[i] = 0xFF
	}
	// Tile 0 MSB plane (all 0s)
	for i := 8; i < 16; i++ {
		chrROM[i] = 0x00
	}

	// Create a mock mapper
	mapper := &mockMapper{
		chrROM:    chrROM,
		mirroring: cartridge.MirrorVertical,
	}

	// Create a cartridge instance
	cart := &cartridge.Cartridge{
		PRGROM: prgROM,
		CHRROM: chrROM,
		Mapper: mapper,
		Mirror: mapper.GetMirroring(),
	}
	return cart
}

// TestPPURenderBackground checks if the PPU correctly renders a solid background tile.
func TestPPURenderBackground(t *testing.T) {
	// Step 1: Initialize PPU and Cartridge
	ppu := New()
	cart := createTestCartridge()
	ppu.ConnectCartridge(cart)

	// Assign a dummy LogDebug function to prevent nil pointer dereference during PPU.Clock()
	LogDebug = func(format string, a ...interface{}) {}

	// Ensure spriteScanline is empty for background-only test
	ppu.spriteScanline = []spriteInfo{}

	// Push all sprites off-screen by initializing OAM Y-coordinates to 0xFF
	for i := 0; i < len(ppu.oam); i++ {
		ppu.oam[i] = 0xFF
	}

	// Manually set initial PPU VRAM for nametable and attribute table
	// We're using Tile ID 0 (solid red) for the entire screen.
	// Nametable 0 (0x2000-0x23FF)
	for i := 0; i < 0x03C0; i++ { // Fill nametable with tile ID 0
		ppu.vram[i] = 0x00
	}
	// Attribute table for nametable 0 (0x23C0-0x23FF)
	// Set all attribute bytes to 0x00 (using palette 0)
	for i := 0x03C0; i < 0x0400; i++ {
		ppu.vram[i] = 0x00
	}

	// Manually set initial PPU Palette
	ppu.palette[0x00] = 0x0F // Universal background color (black)
	ppu.palette[0x01] = 0x16 // Red (from SystemPalette index 0x16)
	ppu.palette[0x02] = 0x20 // Green
	ppu.palette[0x03] = 0x30 // Blue

	// Directly set PPU Control and Mask registers
	ppu.Ctrl = 0x20 // NMI enable, background nametable at $2000
	ppu.Mask = 0x1E // Enable background rendering, enable sprites, show left 8px

	// Step 2: Run PPU Cycles for a few frames
	// 2 frames = 2 * 29780 PPU clocks
	totalPPUCycles := 2 * 89342 // NES PPU cycles per frame: 262 scanlines * 341 cycles/scanline
	for i := 0; i < totalPPUCycles; i++ {
		ppu.Clock()
	}

	// Step 3: Inspect PPU Output (image.RGBA buffer)
	frame := ppu.GetFrame()

	// Expected color: Universal background color is 0x0F (black from SystemPalette).
	// Palette 0, Color 1 is 0x16 (red from SystemPalette).
	// Our tile is rendered with pixel value 1 (color 1), using background palette 0.
	// So, PPURead(0x3F00 + 0*4 + 1) = PPURead(0x3F01) = ppu.palette[1] = 0x16.
	// The SystemPalette[0x16] is {152, 34, 32, 255} (a shade of red).
	expectedColor := ppu.SystemPalette[ppu.palette[1]]

	// Check a few pixels to ensure the background is solid red
	tests := []struct {
		x, y int
	}{
		{0, 0},     // Top-left
		{128, 120}, // Middle
		{255, 239}, // Bottom-right
	}

	for _, tc := range tests {
		actualColor := frame.At(tc.x, tc.y).(color.RGBA)
		if actualColor != expectedColor {
			t.Errorf("At (%d, %d): Expected color %v, got %v", tc.x, tc.y, expectedColor, actualColor)
		}
	}
}
