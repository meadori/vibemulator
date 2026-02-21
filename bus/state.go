package bus

import (
	"encoding/gob"
	"os"

	"github.com/meadori/vibemulator/apu"
	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/cpu"
	"github.com/meadori/vibemulator/ppu"
)

type State struct {
	Ram          [2048]byte
	SystemClocks int
	CPU          cpu.State
	PPU          ppu.State
	APU          apu.State
	Cartridge    cartridge.State
}

// SaveState saves the entire emulator state to a file.
func (b *Bus) SaveState(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	s := State{
		Ram:          b.ram,
		SystemClocks: b.SystemClocks,
		CPU:          b.cpu.SaveState(),
		PPU:          b.PPU.SaveState(),
		APU:          b.APU.SaveState(),
	}

	if b.cart != nil {
		s.Cartridge = b.cart.SaveState()
	}

	return gob.NewEncoder(file).Encode(s)
}

// LoadState loads the emulator state from a file.
func (b *Bus) LoadState(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var s State
	if err := gob.NewDecoder(file).Decode(&s); err != nil {
		return err
	}

	b.ram = s.Ram
	b.SystemClocks = s.SystemClocks
	b.cpu.LoadState(s.CPU)
	b.PPU.LoadState(s.PPU)
	b.APU.LoadState(s.APU)

	if b.cart != nil {
		b.cart.LoadState(s.Cartridge)
	}

	return nil
}
