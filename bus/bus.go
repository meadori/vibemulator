package bus

import (
	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/cpu"
	"github.com/meadori/vibemulator/ppu"
)

// Bus represents the system bus.
type Bus struct {
	cpu *cpu.CPU
	ppu *ppu.PPU
	ram [2048]byte
	cart *cartridge.Cartridge
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
func (b *Bus) LoadCartridge(cart *cartridge.Cartridge) {
	b.cart = cart
	b.ppu.ConnectCartridge(cart)
}

// GetCPU returns the CPU instance.
func (b *Bus) GetCPU() *cpu.CPU {
	return b.cpu
}

func (b *Bus) Clock() {
	b.cpu.Clock()
	b.ppu.Clock()
	b.ppu.Clock()
	b.ppu.Clock()

	if b.ppu.NMI {
		b.ppu.NMI = false
		b.cpu.NMI()
	}
}

// Read reads a byte from the bus.
func (b *Bus) Read(addr uint16) byte {
	if addr >= 0x0000 && addr < 0x2000 {
		return b.ram[addr&0x07FF]
	} else if addr >= 0x2000 && addr < 0x4000 {
		// PPU registers are mirrored every 8 bytes
		return b.ppu.Read(addr & 0x0007)
	} else if addr >= 0x8000 {
		if len(b.cart.PRGROM) == 16384 {
			return b.cart.PRGROM[(addr-0x8000)&0x3FFF]
		}
		return b.cart.PRGROM[addr-0x8000]
	}
	return 0
}

// Write writes a byte to the bus.
func (b *Bus) Write(addr uint16, data byte) {
	if addr >= 0x0000 && addr < 0x2000 {
		b.ram[addr&0x07FF] = data
	} else if addr >= 0x2000 && addr < 0x4000 {
		b.ppu.Write(addr&0x0007, data)
	}
}

