package bus

import (
	"log"

	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/cpu"
	"github.com/meadori/vibemulator/ppu"
)

// Declare logDebug function from main package
var LogDebug func(format string, a ...interface{})

// Bus represents the main bus of the NES.
type Bus struct {
	cpu *cpu.CPU
	PPU *ppu.PPU
	ram [2048]byte

	cart *cartridge.Cartridge

	// SystemClocks keeps track of the total number of clock cycles.
	SystemClocks int
}

// New creates a new Bus instance.
func New() *Bus {
	log.Println("Creating new bus")

	b := &Bus{
		cpu: cpu.New(),
		PPU: ppu.New(),
	}

	b.cpu.ConnectBus(b)

	return b
}

// LoadCartridge loads a cartridge into the bus.
func (b *Bus) LoadCartridge(cart *cartridge.Cartridge) error {
	log.Println("Loading cartridge into bus")
	b.cart = cart
	b.PPU.ConnectCartridge(cart)
	b.cpu.Reset()
	return nil
}

// Clock performs one clock cycle of the system.
func (b *Bus) Clock() {
	b.PPU.Clock()
	// The CPU runs at 1/3 the speed of the PPU
	if b.SystemClocks%3 == 0 {
		if b.PPU.NMI {
			b.PPU.NMI = false
			b.cpu.NMI()
		}
		b.cpu.Clock()
	}

	b.SystemClocks++
}

// Read reads a byte from the bus.
func (b *Bus) Read(addr uint16) byte {
	var data byte
	if data, ok := b.cart.Mapper.CPUMapRead(addr); ok {
		return data
	}

	switch {
	case addr >= 0x0000 && addr <= 0x1FFF:
		data = b.ram[addr&0x07FF]
	case addr >= 0x2000 && addr <= 0x3FFF:
		data = b.PPU.CPURead(addr & 0x0007)
	case addr >= 0x4000 && addr <= 0x4017:
		// APU and I/O registers
	}
	return data
}

// Write writes a byte to the bus.
func (b *Bus) Write(addr uint16, data byte) {
	if ok := b.cart.Mapper.CPUMapWrite(addr, data); ok {
		return
	}

	switch {
	case addr >= 0x0000 && addr <= 0x1FFF:
		b.ram[addr&0x07FF] = data
	case addr >= 0x2000 && addr <= 0x3FFF:
		b.PPU.CPUWrite(addr&0x0007, data)
	case addr == 0x4014:
		// OAMDMA
		b.PPU.DoOAMDMA(data)
	case addr >= 0x4000 && addr <= 0x4017:
		// APU and I/O registers
	}
}
