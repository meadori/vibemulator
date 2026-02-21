package cartridge

import (
	"bytes"
	"encoding/gob"
)

type State struct {
	CHRRAM      []byte
	PRGRAM      []byte // For MMC1/MMC3
	MapperState []byte
}

func (c *Cartridge) SaveState() State {
	s := State{}
	if c.IsCHRRAM {
		s.CHRRAM = make([]byte, len(c.CHRROM))
		copy(s.CHRRAM, c.CHRROM)
	}

	// Dump PRG RAM if the mapper has it
	if m, ok := c.Mapper.(interface{ GetPRGRAM() []byte }); ok {
		ram := m.GetPRGRAM()
		s.PRGRAM = make([]byte, len(ram))
		copy(s.PRGRAM, ram)
	}

	s.MapperState = c.Mapper.Save()
	return s
}

func (c *Cartridge) LoadState(s State) error {
	if c.IsCHRRAM && len(s.CHRRAM) > 0 {
		copy(c.CHRROM, s.CHRRAM)
	}

	// Restore PRG RAM if the mapper has it
	if m, ok := c.Mapper.(interface{ GetPRGRAM() []byte }); ok && len(s.PRGRAM) > 0 {
		ram := m.GetPRGRAM()
		copy(ram, s.PRGRAM)
	}

	return c.Mapper.Load(s.MapperState)
}

// NROM
func (n *nrom) Save() []byte        { return nil }
func (n *nrom) Load(b []byte) error { return nil }

// UXROM
func (u *uxrom) Save() []byte { return []byte{byte(u.prgBankSelect)} }
func (u *uxrom) Load(b []byte) error {
	if len(b) > 0 {
		u.prgBankSelect = int(b[0])
	}
	return nil
}

// CNROM
func (c *cnrom) Save() []byte { return []byte{byte(c.chrBankSelect)} }
func (c *cnrom) Load(b []byte) error {
	if len(b) > 0 {
		c.chrBankSelect = int(b[0])
	}
	return nil
}

// MMC1
type MMC1State struct {
	Control, ChrBank0, ChrBank1, PrgBank, ShiftRegister, WriteCount, WramDisableCounter byte
	WramDisabled                                                                        bool
}

func (m *mmc1) GetPRGRAM() []byte { return m.wram }

func (m *mmc1) Save() []byte {
	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(MMC1State{m.control, m.chrBank0, m.chrBank1, m.prgBank, m.shiftRegister, m.writeCount, m.wramDisableCounter, m.wramDisabled})
	return buf.Bytes()
}

func (m *mmc1) Load(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	var s MMC1State
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&s); err != nil {
		return err
	}
	m.control, m.chrBank0, m.chrBank1, m.prgBank, m.shiftRegister, m.writeCount, m.wramDisableCounter, m.wramDisabled = s.Control, s.ChrBank0, s.ChrBank1, s.PrgBank, s.ShiftRegister, s.WriteCount, s.WramDisableCounter, s.WramDisabled
	return nil
}

// MMC3
type MMC3State struct {
	TargetRegister                                         byte
	PrgBankMode, ChrInversion                              bool
	Registers                                              [8]byte
	IrqCounter, IrqLatch                                   byte
	IrqReload, IrqEnabled, IrqPending, LastA12, FourScreen bool
	A12Delay                                               int
	Mirroring                                              byte
}

func (m *mmc3) GetPRGRAM() []byte { return m.prgRAM }

func (m *mmc3) Save() []byte {
	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(MMC3State{m.targetRegister, m.prgBankMode, m.chrInversion, m.registers, m.irqCounter, m.irqLatch, m.irqReload, m.irqEnabled, m.irqPending, m.lastA12, m.fourScreen, m.a12Delay, m.mirroring})
	return buf.Bytes()
}

func (m *mmc3) Load(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	var s MMC3State
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&s); err != nil {
		return err
	}
	m.targetRegister, m.prgBankMode, m.chrInversion, m.registers, m.irqCounter, m.irqLatch, m.irqReload, m.irqEnabled, m.irqPending, m.lastA12, m.fourScreen, m.a12Delay, m.mirroring = s.TargetRegister, s.PrgBankMode, s.ChrInversion, s.Registers, s.IrqCounter, s.IrqLatch, s.IrqReload, s.IrqEnabled, s.IrqPending, s.LastA12, s.FourScreen, s.A12Delay, s.Mirroring
	return nil
}
