package cartridge

// cnrom represents Mapper 3 (CNROM).
// It features fixed PRG ROM (16KB or 32KB) and switchable 8KB CHR ROM banks.
// Bank switching is done by writing to any address in $8000-$FFFF.
type cnrom struct {
	prgROM        []byte
	chrROM        []byte
	mirror        byte
	prgBanks      int
	chrBanks      int
	chrBankSelect int
}

func newCNROM(cart *Cartridge) *cnrom {
	prgBanks := len(cart.PRGROM) / 16384
	chrBanks := len(cart.CHRROM) / 8192
	return &cnrom{
		prgROM:        cart.PRGROM,
		chrROM:        cart.CHRROM,
		mirror:        cart.Mirror,
		prgBanks:      prgBanks,
		chrBanks:      chrBanks,
		chrBankSelect: 0,
	}
}

// CPUMapRead implements the Mapper interface for CPU reads.
func (c *cnrom) CPUMapRead(addr uint16) (byte, bool) {
	if addr >= 0x8000 && addr <= 0xFFFF {
		mappedAddr := addr - 0x8000
		if c.prgBanks == 1 { // 16KB PRG ROM, mirror it in the upper 16KB
			mappedAddr &= 0x3FFF
		}
		return c.prgROM[mappedAddr], true
	}
	return 0, false
}

// CPUMapWrite implements the Mapper interface for CPU writes.
func (c *cnrom) CPUMapWrite(addr uint16, data byte) bool {
	if addr >= 0x8000 && addr <= 0xFFFF {
		// CHR bank select is written to $8000-$FFFF
		if c.chrBanks > 0 {
			c.chrBankSelect = int(data) % c.chrBanks
		}
		return true
	}
	return false
}

// PPUMapRead implements the Mapper interface for PPU reads.
func (c *cnrom) PPUMapRead(addr uint16) (byte, bool) {
	if addr <= 0x1FFF {
		mappedAddr := (c.chrBankSelect * 8192) + int(addr)
		return c.chrROM[mappedAddr], true
	}
	return 0, false
}

// PPUMapWrite implements the Mapper interface for PPU writes.
func (c *cnrom) PPUMapWrite(addr uint16, data byte) bool {
	if addr <= 0x1FFF {
		// CNROM is typically CHR-ROM, but handle CHR-RAM just in case
		if len(c.chrROM) == 8192 {
			mappedAddr := (c.chrBankSelect * 8192) + int(addr)
			c.chrROM[mappedAddr] = data
			return true
		}
	}
	return false
}

// GetMirroring implements the Mapper interface to return the cartridge's mirroring type.
func (c *cnrom) GetMirroring() byte {
	return c.mirror
}

// Clock ticks the mapper (no-op for CNROM).
func (c *cnrom) Clock() {}
