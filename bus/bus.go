package bus

import (
	"log"

	"github.com/meadori/vibemulator/apu"
	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/controller"
	"github.com/meadori/vibemulator/cpu"
	"github.com/meadori/vibemulator/ppu"
)

// Declare logDebug function from main package
var LogDebug func(format string, a ...interface{})

// Bus represents the main bus of the NES.
type Bus struct {
	cpu  *cpu.CPU
	PPU  *ppu.PPU
	APU  *apu.APU
	ram  [2048]byte
	cart *cartridge.Cartridge
	joy1 *controller.Controller
	joy2 *controller.Controller

	// SystemClocks keeps track of the total number of clock cycles.
	SystemClocks int
}

// New creates a new Bus instance.
func New() *Bus {
	log.Println("Creating new bus")

	b := &Bus{
		cpu:  cpu.New(),
		PPU:  ppu.New(),
		APU:  apu.New(),
		joy1: controller.New(),
		joy2: controller.New(),
	}

	b.cpu.ConnectBus(b)
	b.APU.ConnectBus(b)

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

// EjectCartridge removes the cartridge from the bus.
func (b *Bus) EjectCartridge() {
	log.Println("Ejecting cartridge from bus")
	b.PowerOff()
	b.cart = nil
	b.PPU.ConnectCartridge(nil)
}

// PowerOff silences the system and resets internal state but keeps the cartridge.
func (b *Bus) PowerOff() {
	log.Println("Powering off bus")
	b.APU.CPUWrite(0x4015, 0) // Disable all sound channels
	b.PPU.Reset()
	// Clear internal RAM
	for i := range b.ram {
		b.ram[i] = 0
	}
}

// PowerOn resets the system components to start execution.
func (b *Bus) PowerOn() {
	log.Println("Powering on bus")
	b.PPU.Reset()
	b.cpu.Reset()
}

// HasCartridge returns true if a cartridge is currently loaded.
func (b *Bus) HasCartridge() bool {
	return b.cart != nil
}

// Clock performs one clock cycle of the system.
func (b *Bus) Clock() {
	b.PPU.Clock()
	// The CPU runs at 1/3 the speed of the PPU
	if b.SystemClocks%3 == 0 {
		// Clock APU first to ensure IRQ status is updated for current CPU cycle
		b.APU.Clock()
		if b.cart != nil {
			b.cart.Mapper.Clock()
		}
		// Check for NMI (PPU)
		if b.PPU.NMI {
			b.PPU.NMI = false
			b.cpu.NMI()
		}

		// Check for APU IRQ (DMC or Frame IRQ)
		cartIRQ := false
		if b.cart != nil {
			cartIRQ = b.cart.Mapper.IRQPending()
		}
		if b.APU.DmcIRQ || b.APU.FrameIRQ || cartIRQ {
			b.cpu.IRQ()
		}

		b.cpu.Clock() // Clock the CPU after all IRQ checks
	}

	b.SystemClocks++
}

// GetFramePixels returns the raw PPU frame buffer for the RL Agent
func (b *Bus) GetFramePixels() []byte {
	return b.PPU.GetFrame().Pix
}

// Read reads a byte from the bus.
func (b *Bus) Read(addr uint16) byte {
	var data byte
	if b.cart != nil {
		if data, ok := b.cart.Mapper.CPUMapRead(addr); ok {
			return data
		}
	}

	switch {
	case addr >= 0x0000 && addr <= 0x1FFF:
		data = b.ram[addr&0x07FF]
	case addr >= 0x2000 && addr <= 0x3FFF:
		data = b.PPU.CPURead(addr & 0x0007)
	case addr == 0x4016:
		data = b.joy1.Read()
	case addr == 0x4017:
		data = b.joy2.Read()
	case addr >= 0x4000 && addr <= 0x4017:
		data = b.APU.CPURead(addr)
	}
	return data
}

// Write writes a byte to the bus.
func (b *Bus) Write(addr uint16, data byte) {
	if b.cart != nil {
		if ok := b.cart.Mapper.CPUMapWrite(addr, data); ok {
			return
		}
	}

	switch {
	case addr >= 0x0000 && addr <= 0x1FFF:
		b.ram[addr&0x07FF] = data
	case addr >= 0x2000 && addr <= 0x3FFF:
		b.PPU.CPUWrite(addr&0x0007, data)
	case addr == 0x4014:
		// OAMDMA
		oamData := [256]byte{}
		dmaAddr := uint16(data) << 8
		for i := 0; i < 256; i++ {
			oamData[i] = b.Read(dmaAddr + uint16(i))
		}
		b.PPU.DoOAMDMA(oamData)
	case addr == 0x4016:
		b.joy1.Write(data)
		b.joy2.Write(data)
	case addr >= 0x4000 && addr <= 0x4017:
		b.APU.CPUWrite(addr, data)
	}
}

// SetController1State sets the state of the buttons for controller 1.
func (b *Bus) SetController1State(buttons [8]bool) {
	b.joy1.SetButtons(buttons)
}

// SetController2State sets the state of the buttons for controller 2.
func (b *Bus) SetController2State(buttons [8]bool) {
	b.joy2.SetButtons(buttons)
}

func (b *Bus) Reset() {
	b.cpu.Reset()
}
