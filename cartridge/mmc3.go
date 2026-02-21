package cartridge

// mmc3 represents Mapper 4 (MMC3).
// It features complex PRG and CHR bank switching and a scanline-based IRQ counter.
type mmc3 struct {
	prgROM []byte
	chrROM []byte
	prgRAM []byte
	chrRAM bool

	targetRegister byte
	prgBankMode    bool // false: $8000 is swappable, true: $C000 is swappable
	chrInversion   bool // false: two 2KB banks at $0000, true: two 2KB banks at $1000
	registers      [8]byte

	prgBanks int
	chrBanks int

	// IRQ State
	irqCounter byte
	irqLatch   byte
	irqReload  bool
	irqEnabled bool
	irqPending bool
	lastA12    bool
	a12Delay   int
	fourScreen bool
	mirroring  byte
}

func newMMC3(cart *Cartridge) *mmc3 {
	prgBanks := len(cart.PRGROM) / 8192
	chrBanks := len(cart.CHRROM) / 1024

	// Handle 4-screen mirroring flag
	fourScreen := (cart.Mirror & 4) != 0
	mirroring := cart.Mirror & 1

	return &mmc3{
		prgROM:     cart.PRGROM,
		chrROM:     cart.CHRROM,
		prgRAM:     make([]byte, 8192),
		chrRAM:     cart.IsCHRRAM,
		prgBanks:   prgBanks,
		chrBanks:   chrBanks,
		fourScreen: fourScreen,
		mirroring:  mirroring,
	}
}

// CPUMapRead implements the Mapper interface for CPU reads.
func (m *mmc3) CPUMapRead(addr uint16) (byte, bool) {
	if addr >= 0x6000 && addr <= 0x7FFF {
		return m.prgRAM[addr-0x6000], true
	} else if addr >= 0x8000 && addr <= 0xFFFF {
		bank := m.getPRGBank(addr)
		mappedAddr := (bank * 8192) + int(addr&0x1FFF)
		return m.prgROM[mappedAddr], true
	}
	return 0, false
}

func (m *mmc3) getPRGBank(addr uint16) int {
	secondToLast := m.prgBanks - 2
	last := m.prgBanks - 1

	if addr >= 0x8000 && addr <= 0x9FFF {
		if m.prgBankMode {
			return secondToLast
		}
		return int(m.registers[6]) % m.prgBanks
	} else if addr >= 0xA000 && addr <= 0xBFFF {
		return int(m.registers[7]) % m.prgBanks
	} else if addr >= 0xC000 && addr <= 0xDFFF {
		if m.prgBankMode {
			return int(m.registers[6]) % m.prgBanks
		}
		return secondToLast
	} else if addr >= 0xE000 && addr <= 0xFFFF {
		return last
	}
	return 0
}

// CPUMapWrite implements the Mapper interface for CPU writes.
func (m *mmc3) CPUMapWrite(addr uint16, data byte) bool {
	if addr >= 0x6000 && addr <= 0x7FFF {
		m.prgRAM[addr-0x6000] = data
		return true
	}

	if addr >= 0x8000 && addr <= 0xFFFF {
		isEven := (addr % 2) == 0

		switch {
		case addr >= 0x8000 && addr <= 0x9FFF:
			if isEven {
				m.targetRegister = data & 0x07
				m.prgBankMode = (data & 0x40) != 0
				m.chrInversion = (data & 0x80) != 0
			} else {
				m.registers[m.targetRegister] = data
			}
		case addr >= 0xA000 && addr <= 0xBFFF:
			if isEven {
				m.mirroring = data & 1
			} else {
				// PRG RAM protect (ignored in basic implementation)
			}
		case addr >= 0xC000 && addr <= 0xDFFF:
			if isEven {
				m.irqLatch = data
			} else {
				m.irqReload = true
			}
		case addr >= 0xE000 && addr <= 0xFFFF:
			if isEven {
				m.irqEnabled = false
				m.irqPending = false
			} else {
				m.irqEnabled = true
			}
		}
		return true
	}
	return false
}

// PPUMapRead implements the Mapper interface for PPU reads.
func (m *mmc3) PPUMapRead(addr uint16) (byte, bool) {
	if addr <= 0x1FFF {
		m.checkA12(addr)
		bank := m.getCHRBank(addr)
		mappedAddr := (bank * 1024) + int(addr&0x03FF)
		return m.chrROM[mappedAddr], true
	}
	return 0, false
}

// PPUMapWrite implements the Mapper interface for PPU writes.
func (m *mmc3) PPUMapWrite(addr uint16, data byte) bool {
	if addr <= 0x1FFF {
		m.checkA12(addr)
		if m.chrRAM {
			bank := m.getCHRBank(addr)
			mappedAddr := (bank * 1024) + int(addr&0x03FF)
			m.chrROM[mappedAddr] = data
			return true
		}
	}
	return false
}

func (m *mmc3) getCHRBank(addr uint16) int {
	if m.chrInversion {
		switch {
		case addr <= 0x03FF:
			return int(m.registers[2]) % m.chrBanks
		case addr <= 0x07FF:
			return int(m.registers[3]) % m.chrBanks
		case addr <= 0x0BFF:
			return int(m.registers[4]) % m.chrBanks
		case addr <= 0x0FFF:
			return int(m.registers[5]) % m.chrBanks
		case addr <= 0x13FF:
			return int(m.registers[0]&0xFE) % m.chrBanks
		case addr <= 0x17FF:
			return int((m.registers[0]&0xFE)|1) % m.chrBanks
		case addr <= 0x1BFF:
			return int(m.registers[1]&0xFE) % m.chrBanks
		case addr <= 0x1FFF:
			return int((m.registers[1]&0xFE)|1) % m.chrBanks
		}
	} else {
		switch {
		case addr <= 0x03FF:
			return int(m.registers[0]&0xFE) % m.chrBanks
		case addr <= 0x07FF:
			return int((m.registers[0]&0xFE)|1) % m.chrBanks
		case addr <= 0x0BFF:
			return int(m.registers[1]&0xFE) % m.chrBanks
		case addr <= 0x0FFF:
			return int((m.registers[1]&0xFE)|1) % m.chrBanks
		case addr <= 0x13FF:
			return int(m.registers[2]) % m.chrBanks
		case addr <= 0x17FF:
			return int(m.registers[3]) % m.chrBanks
		case addr <= 0x1BFF:
			return int(m.registers[4]) % m.chrBanks
		case addr <= 0x1FFF:
			return int(m.registers[5]) % m.chrBanks
		}
	}
	return 0
}

func (m *mmc3) checkA12(addr uint16) {
	a12 := (addr & 0x1000) != 0

	// Trigger on rising edge of A12, but only if it was low for a while
	if a12 && !m.lastA12 && m.a12Delay >= 2 {
		m.clockIRQ()
	}

	if a12 {
		m.lastA12 = true
		m.a12Delay = 0
	} else {
		m.lastA12 = false
	}
}

func (m *mmc3) clockIRQ() {
	if m.irqCounter == 0 || m.irqReload {
		m.irqCounter = m.irqLatch
		m.irqReload = false
	} else {
		m.irqCounter--
	}

	if m.irqCounter == 0 && m.irqEnabled {
		m.irqPending = true
	}
}

// GetMirroring implements the Mapper interface to return the cartridge's mirroring type.
func (m *mmc3) GetMirroring() byte {
	if m.fourScreen {
		return 4 // MirrorFourScreen
	}
	if m.mirroring == 0 {
		return 1 // MirrorVertical
	}
	return 0 // MirrorHorizontal
}

func (m *mmc3) Clock() {
	// Increment a12 delay if a12 is currently low (checked every CPU cycle)
	if !m.lastA12 {
		m.a12Delay++
	}
}

func (m *mmc3) IRQPending() bool {
	return m.irqPending
}

func (m *mmc3) ClearIRQ() {
	m.irqPending = false
}

// PPUDebugRead implements a side-effect free PPU read for the PPU Debugger overlay, skipping the A12 IRQ counter update.
func (m *mmc3) PPUDebugRead(addr uint16) (byte, bool) {
	if addr <= 0x1FFF {
		bank := m.getCHRBank(addr)
		mappedAddr := (bank * 1024) + int(addr&0x03FF)
		return m.chrROM[mappedAddr], true
	}
	return 0, false
}
