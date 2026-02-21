package cartridge

// uxrom represents Mapper 2 (UxROM).
// It features a switchable 16KB PRG ROM bank at $8000-$BFFF
// and a fixed 16KB PRG ROM bank at $C000-$FFFF (the last bank).
// It typically uses 8KB of CHR-RAM, which is unbanked.
type uxrom struct {
	prgROM        []byte
	chrROM        []byte
	mirror        byte
	prgBanks      int
	prgBankSelect int
}

func newUxROM(cart *Cartridge) *uxrom {
	prgBanks := len(cart.PRGROM) / 16384
	return &uxrom{
		prgROM:        cart.PRGROM,
		chrROM:        cart.CHRROM,
		mirror:        cart.Mirror,
		prgBanks:      prgBanks,
		prgBankSelect: 0,
	}
}

// CPUMapRead implements the Mapper interface for CPU reads.
func (u *uxrom) CPUMapRead(addr uint16) (byte, bool) {
	if addr >= 0x8000 && addr <= 0xBFFF {
		// Switchable 16KB bank
		bank := u.prgBankSelect % u.prgBanks
		mappedAddr := (bank * 16384) + int(addr-0x8000)
		return u.prgROM[mappedAddr], true
	} else if addr >= 0xC000 && addr <= 0xFFFF {
		// Fixed last 16KB bank
		bank := u.prgBanks - 1
		mappedAddr := (bank * 16384) + int(addr-0xC000)
		return u.prgROM[mappedAddr], true
	}
	return 0, false
}

// CPUMapWrite implements the Mapper interface for CPU writes.
func (u *uxrom) CPUMapWrite(addr uint16, data byte) bool {
	if addr >= 0x8000 && addr <= 0xFFFF {
		// Bank select written to any address in $8000-$FFFF
		u.prgBankSelect = int(data)
		return true
	}
	return false
}

// PPUMapRead implements the Mapper interface for PPU reads.
func (u *uxrom) PPUMapRead(addr uint16) (byte, bool) {
	if addr <= 0x1FFF {
		return u.chrROM[addr], true
	}
	return 0, false
}

// PPUMapWrite implements the Mapper interface for PPU writes.
func (u *uxrom) PPUMapWrite(addr uint16, data byte) bool {
	if addr <= 0x1FFF {
		// Most UxROM boards use CHR-RAM, but we check if we allocated it as RAM
		if len(u.chrROM) == 8192 {
			u.chrROM[addr] = data
			return true
		}
	}
	return false
}

// GetMirroring implements the Mapper interface to return the cartridge's mirroring type.
func (u *uxrom) GetMirroring() byte {
	return u.mirror
}

// Clock ticks the mapper (no-op for UxROM).
func (u *uxrom) Clock() {}

func (u *uxrom) IRQPending() bool { return false }
func (u *uxrom) ClearIRQ()        {}
