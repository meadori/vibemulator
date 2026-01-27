package bus

import (
	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/cpu"
	"github.com/meadori/vibemulator/mapper" // Import mapper package
	"github.com/meadori/vibemulator/ppu"
)

// Bus represents the system bus.
type Bus struct {
	cpu *cpu.CPU
	ppu *ppu.PPU
	ram [2048]byte
	
	cart *cartridge.Cartridge // Keep cartridge for now, might be useful
	mapper mapper.Mapper      // New mapper field
	dmaTransfer int
}

// New creates a new Bus instance.
func New() *Bus {
	bus := &Bus{
		cpu: cpu.New(),
		ppu: ppu.New(),
	}
	bus.cpu.ConnectBus(bus)
	return bus
}

// LoadCartridge loads a cartridge into the bus.
func (b *Bus) LoadCartridge(cart *cartridge.Cartridge) error { // Added error return
	b.cart = cart
	
	// Create mapper for the cartridge
	newMapper, err := mapper.NewMapper(cart)
	if err != nil {
		return err
	}
	b.mapper = newMapper
	
	// PPU also needs to connect to the mapper for CHR access and mirroring
	// b.ppu.ConnectCartridge(cart) // PPU should now connect to mapper instead of raw cartridge
	b.ppu.ConnectMapper(b.mapper) // New: PPU connects to mapper

	return nil // No error
}

// GetPPU returns the PPU instance.
func (b *Bus) GetPPU() *ppu.PPU {
	return b.ppu
}

// GetCPU returns the CPU instance.
func (b *Bus) GetCPU() *cpu.CPU {
	return b.cpu
}

func (b *Bus) Clock() {
	b.ppu.Clock()
	b.ppu.Clock()
	b.ppu.Clock()

	if b.dmaTransfer > 0 {
		b.dmaTransfer--
		if b.dmaTransfer == 0 {
			// DMA finished, CPU can resume
		}
	} else {
		b.cpu.Clock()
	}

	if b.ppu.NMI {
		b.ppu.NMI = false
		b.cpu.NMI()
	}
}

// Read reads a byte from the bus.
func (b *Bus) Read(addr uint16) byte {
	if addr >= 0x0000 && addr < 0x2000 { // 2KB internal RAM, mirrored
		return b.ram[addr&0x07FF]
	} else if addr >= 0x2000 && addr < 0x4000 { // PPU registers, mirrored
		return b.ppu.Read(addr & 0x0007)
	} else if addr >= 0x4000 && addr < 0x4020 { // APU and I/O registers
		// TODO: Implement APU and I/O registers
		return 0x00 // Open bus
	} else if addr >= 0x4020 && addr < 0x6000 { // Expansion ROM (usually unused)
		// TODO: Mapper specific expansion ROM
		return 0x00 // Open bus
	} else if addr >= 0x6000 && addr <= 0xFFFF { // PRG-RAM, PRG-ROM
		if b.mapper != nil {
			data, handled := b.mapper.CPUMapRead(addr)
			if handled {
				return data
			}
		}
		// If mapper didn't handle it, assume open bus or unhandled RAM
		// For NROM, 0x6000-0x7FFF is typically unhandled WRAM that can be used by the program
		if addr >= 0x6000 && addr < 0x8000 {
			// This region can sometimes contain battery-backed RAM or trainer data.
			// For NROM, we generally treat it as open bus if the mapper doesn't explicitly handle it.
			return 0x00 
		}
	}
	return 0x00 // Unhandled address, open bus
}

// Write writes a byte to the bus.
func (b *Bus) Write(addr uint16, data byte) {
	if addr >= 0x0000 && addr < 0x2000 { // 2KB internal RAM, mirrored
		b.ram[addr&0x07FF] = data
	} else if addr >= 0x2000 && addr < 0x4000 { // PPU registers, mirrored
		b.ppu.Write(addr&0x0007, data)
	} else if addr >= 0x4000 && addr < 0x4020 { // APU and I/O registers
		if addr == 0x4014 { // OAMDMA
			b.ppu.DoOAMDMA(data, b.Read)
			if b.cpu.Cycles%2 == 0 { // Check CPU cycle parity
				b.dmaTransfer = 513
			} else {
				b.dmaTransfer = 514
			}
		}
		// TODO: Implement other APU and I/O registers
	} else if addr >= 0x4020 && addr < 0x6000 { // Expansion ROM (usually unused)
		// TODO: Mapper specific
	} else if addr >= 0x6000 && addr <= 0xFFFF { // PRG-RAM, PRG-ROM
		if b.mapper != nil {
			if b.mapper.CPUMapWrite(addr, data) {
				return // Handled by mapper
			}
		}
		// If mapper didn't handle it, assume unhandled RAM
		if addr >= 0x6000 && addr < 0x8000 {
			// Open bus or unhandled WRAM
			return
		}
	}
}
