package cartridge

// MMC1 (Mapper 1) is a common mapper that supports bank switching.
type mmc1 struct {
	prgROM []byte
	chrROM []byte
	wram   []byte
	chrRAM bool
	cart   *Cartridge
	wramDisabled bool
	wramDisableCounter byte

	// Registers
	control    byte
	chrBank0   byte
	chrBank1   byte
	prgBank    byte

	// Shift register for serial writes
	shiftRegister byte
	writeCount    byte
}

func newMMC1(cart *Cartridge) *mmc1 {
	return &mmc1{
		prgROM:  cart.PRGROM,
		chrROM:  cart.CHRROM,
		wram:    make([]byte, 8192),
		control: 0x0C,
		chrRAM:  cart.IsCHRRAM,
		cart:    cart,
	}
}

// CPUMapRead implements the Mapper interface for CPU reads.
func (m *mmc1) CPUMapRead(addr uint16) (byte, bool) {
	if addr >= 0x6000 && addr <= 0x7FFF {
		// WRAM
		if !m.wramDisabled {
			return m.wram[addr-0x6000], true
		}
		return 0, false
	} else if addr >= 0x8000 && addr <= 0xFFFF {
		prgBankMode := (m.control >> 2) & 3
		numPrgBanks := uint32(len(m.prgROM) / 16384)

		var finalAddr uint32
		switch prgBankMode {
		case 0, 1: // switch 32 KB at $8000
			bank := uint32(m.prgBank & 0x0E) >> 1
			bank %= (numPrgBanks / 2)
			finalAddr = bank*32768 + uint32(addr&0x7FFF)
		case 2: // fix first bank at $8000 and switch 16 KB bank at $C000
			var bank uint32
			if addr < 0xC000 {
				bank = 0
			} else {
				bank = uint32(m.prgBank & 0x0F)
				bank %= numPrgBanks
			}
			finalAddr = bank*16384 + uint32(addr&0x3FFF)
		case 3: // fix last bank at $C000 and switch 16 KB bank at $8000
			var bank uint32
			if addr < 0xC000 {
				bank = uint32(m.prgBank & 0x0F)
				bank %= numPrgBanks
			} else {
				bank = numPrgBanks - 1
			}
			finalAddr = bank*16384 + uint32(addr&0x3FFF)
		}
		return m.prgROM[finalAddr], true
	}
	return 0, false
}

func (m *mmc1) CPUMapWrite(addr uint16, data byte) bool {
	if addr >= 0x6000 && addr <= 0x7FFF {
		// WRAM
		if !m.wramDisabled {
			m.wram[addr-0x6000] = data
			return true
		}
		return false
	} else if addr >= 0x8000 && addr <= 0xFFFF {
		if data&0x80 == 1 {
			m.shiftRegister = 0
			m.writeCount = 0
			// Also reset control register
			m.control |= 0x0C
			return true
		}

		m.shiftRegister = (m.shiftRegister >> 1) | ((data & 1) << 4)
		m.writeCount++

		if m.writeCount == 5 {
			targetRegister := (addr >> 13) & 3
			switch targetRegister {
			case 0: // Control
				m.control = m.shiftRegister
				m.cart.Mirror = m.GetMirroring()
			case 1: // CHR bank 0
				m.chrBank0 = m.shiftRegister
			case 2: // CHR bank 1
				m.chrBank1 = m.shiftRegister
			case 3: // PRG bank
				m.prgBank = m.shiftRegister
				if (m.prgBank>>4)&1 == 1 {
					m.wramDisableCounter = 2
				} else {
					m.wramDisabled = false
				}
			}
			m.shiftRegister = 0
			m.writeCount = 0
		}
		return true
	}
	return false
}

// PPUMapRead implements the Mapper interface for PPU reads.
func (m *mmc1) PPUMapRead(addr uint16) (byte, bool) {
	if addr >= 0x0000 && addr <= 0x1FFF {
		chrBankMode := (m.control >> 4) & 1
		numChrBanks := uint32(len(m.chrROM) / 4096)

		var finalAddr uint32

		if chrBankMode == 0 { // 8KB mode
			bank := uint32(m.chrBank0 & 0x1E) >> 1
			bank %= (numChrBanks / 2)
			finalAddr = bank*8192 + uint32(addr&0x1FFF)
		} else { // 4KB mode
			var bank uint32
			if addr < 0x1000 {
				bank = uint32(m.chrBank0)
			} else {
				bank = uint32(m.chrBank1)
			}
			bank %= numChrBanks
			finalAddr = bank*4096 + uint32(addr&0x0FFF)
		}

		return m.chrROM[finalAddr], true
	}
	return 0, false
}

// PPUMapWrite implements the Mapper interface for PPU writes.
func (m *mmc1) PPUMapWrite(addr uint16, data byte) bool {
	if addr >= 0x0000 && addr <= 0x1FFF {
		// Only allow writes if it's CHR-RAM
		if m.chrRAM {
			chrBankMode := (m.control >> 4) & 1
			numChrBanks := uint32(len(m.chrROM) / 4096)

			var finalAddr uint32

			if chrBankMode == 0 { // 8KB mode
				bank := uint32(m.chrBank0 & 0x1E) >> 1
				bank %= (numChrBanks / 2)
				finalAddr = bank*8192 + uint32(addr&0x1FFF)
			} else { // 4KB mode
				var bank uint32
				if addr < 0x1000 {
					bank = uint32(m.chrBank0)
				} else {
					bank = uint32(m.chrBank1)
				}
				bank %= numChrBanks
				finalAddr = bank*4096 + uint32(addr&0x0FFF)
			}

			m.chrROM[finalAddr] = data
			return true
		}
	}
	return false
}

// GetMirroring implements the Mapper interface to return the cartridge's mirroring type.
func (m *mmc1) GetMirroring() byte {
	switch m.control & 3 {
	case 0:
		return 2 // One-screen, lower bank
	case 1:
		return 3 // One-screen, upper bank
	case 2:
		return 1 // Vertical
	case 3:
		return 0 // Horizontal
	}
	return 0
}

func (m *mmc1) Clock() {
	if m.wramDisableCounter > 0 {
		m.wramDisableCounter--
		if m.wramDisableCounter == 0 {
			m.wramDisabled = true
		}
	}
}
