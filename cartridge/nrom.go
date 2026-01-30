package cartridge

// NROM (Mapper 0) is the simplest mapper.
type nrom struct {
	prgROM   []byte
	chrROM   []byte
	mirror   byte
	prgBanks int // 1 or 2 (16KB or 32KB)
	chrBanks int // 1 or 2 (8KB or 16KB), or 0 if CHR-RAM was allocated.
}

func newNROM(cart *Cartridge) *nrom {
	prgBanks := len(cart.PRGROM) / 16384 // 16KB banks
	chrBanks := len(cart.CHRROM) / 8192  // 8KB banks (note: if CHR-RAM, len(CHRROM) will be 8192 and chrBanks 1)

	return &nrom{
		prgROM:   cart.PRGROM,
		chrROM:   cart.CHRROM,
		mirror:   cart.Mirror,
		prgBanks: prgBanks,
		chrBanks: chrBanks,
	}
}

// CPUMapRead implements the Mapper interface for CPU reads.
func (n *nrom) CPUMapRead(addr uint16) (byte, bool) {
	if addr >= 0x6000 && addr <= 0x7FFF {
		// NROM typically does not have PRG-RAM mapped here, or it's a fixed value.
		// For now, treat as unhandled by mapper. Bus's RAM will handle it if needed.
		return 0, false
	} else if addr >= 0x8000 && addr <= 0xFFFF {
		// PRG-ROM
		mappedAddr := addr - 0x8000
		if n.prgBanks == 1 { // 16KB PRG ROM, mirror it in the upper 16KB
			mappedAddr &= 0x3FFF // Maps 0x8000-0xBFFF and 0xC000-0xFFFF to the 16KB PRG ROM
		}
		return n.prgROM[mappedAddr], true
	}
	return 0, false // Address not handled by NROM mapper
}

// CPUMapWrite implements the Mapper interface for CPU writes.
func (n *nrom) CPUMapWrite(addr uint16, data byte) bool {
	// NROM usually doesn't have CPU-writable memory through the mapper (e.g., PRG-RAM).
	// Any WRAM would be handled by the main Bus RAM.
	return false
}

// PPUMapRead implements the Mapper interface for PPU reads.
func (n *nrom) PPUMapRead(addr uint16) (byte, bool) {
	if addr >= 0x0000 && addr <= 0x1FFF {
		// CHR-ROM or CHR-RAM
		// NROM usually maps the 8KB CHR-ROM/RAM directly.
		return n.chrROM[addr], true
	}
	return 0, false // Address not handled by NROM mapper
}

// PPUMapWrite implements the Mapper interface for PPU writes.
func (n *nrom) PPUMapWrite(addr uint16, data byte) bool {
	if addr >= 0x0000 && addr <= 0x1FFF {
		// Only allow writes if it's CHR-RAM (CHR-ROM is read-only).
		// We're assuming if CHRROM len is 8192, it's CHR-RAM (based on cartridge.go allocating 8192 bytes for CHR-RAM).
		if len(n.chrROM) == 8192 { 
			n.chrROM[addr] = data
			return true
		}
	}
	return false // Address not handled by NROM mapper, or it's CHR-ROM (read-only)
}

// GetMirroring implements the Mapper interface to return the cartridge's mirroring type.
func (n *nrom) GetMirroring() byte {
	return n.mirror
}
