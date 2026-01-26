package ppu

import "github.com/meadori/vibemulator/cartridge"

// PPU represents the Picture Processing Unit.
type PPU struct {
	cart *cartridge.Cartridge
	vram [2048]byte
	oam  [256]byte
	palette [32]byte

	// PPU Control Register
	PPUCTRL byte
	// PPU Mask Register
	PPUMASK byte
	// PPU Status Register
	PPUSTATUS byte
	// OAM Address Register
	OAMADDR byte
	// OAM Data Register
	OAMDATA byte
	// PPU Scroll Register
	PPUSCROLL byte
	// PPU Address Register
	PPUADDR byte
	// PPU Data Register
	PPUDATA byte
	// OAM DMA Register
	OAMDMA byte

	NMI bool

	vramAddr uint16
	vramTmpAddr uint16
	dataBuffer byte
	addrLatch bool
	fineX byte

	scanline int
	cycle int

	bgNextTileID     byte
	bgNextTileAttrib byte
	bgNextTileLSB    byte
	bgNextTileMSB    byte
	bgShifterPatternL uint16
	bgShifterPatternH uint16
	bgShifterAttribL  uint16
	bgShifterAttribH  uint16

	pixels [256 * 240]byte
}

// New creates a new PPU instance.
func New() *PPU {
	return &PPU{}
}

// ConnectCartridge connects a cartridge to the PPU.
func (p *PPU) ConnectCartridge(cart *cartridge.Cartridge) {
	p.cart = cart
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func (p *PPU) ppuWrite(addr uint16, data byte) {
	addr &= 0x3FFF
	if addr >= 0x0000 && addr <= 0x1FFF {
		// CHR ROM is usually read-only
	} else if addr >= 0x2000 && addr <= 0x3EFF {
		p.vram[addr&0x0FFF] = data
	} else if addr >= 0x3F00 && addr <= 0x3FFF {
		addr &= 0x001F
		if addr == 0x0010 {
			addr = 0x0000
		}
		if addr == 0x0014 {
			addr = 0x0004
		}
		if addr == 0x0018 {
			addr = 0x0008
		}
		if addr == 0x001C {
			addr = 0x000C
		}
		p.palette[addr] = data
	}
}

func (p *PPU) ppuRead(addr uint16) byte {
	addr &= 0x3FFF
	if addr >= 0x0000 && addr <= 0x1FFF {
		return p.cart.CHRROM[addr]
	} else if addr >= 0x2000 && addr <= 0x3EFF {
		return p.vram[addr&0x0FFF]
	} else if addr >= 0x3F00 && addr <= 0x3FFF {
		addr &= 0x001F
		if addr == 0x0010 {
			addr = 0x0000
		}
		if addr == 0x0014 {
			addr = 0x0004
		}
		if addr == 0x0018 {
			addr = 0x0008
		}
		if addr == 0x001C {
			addr = 0x000C
		}
		return p.palette[addr]
	}
	return 0
}

func (p *PPU) GetPixels() []byte {
	return p.pixels[:]
}

func (p *PPU) Clock() {
	if p.scanline >= -1 && p.scanline < 240 {
		if p.scanline == -1 && p.cycle == 1 {
			p.PPUSTATUS &^= 0x80
		}

		if p.scanline >= 0 && p.cycle < 256 {
			var paletteIdx byte
			if p.PPUMASK&(1<<3) != 0 {
				mux := 0x8000 >> uint(p.fineX)
				p1 := (p.bgShifterPatternH & uint16(mux)) > 0
				p0 := (p.bgShifterPatternL & uint16(mux)) > 0
				paletteIdx = (boolToByte(p1) << 1) | boolToByte(p0)

				a1 := (p.bgShifterAttribH & uint16(mux)) > 0
				a0 := (p.bgShifterAttribL & uint16(mux)) > 0
				paletteIdx |= ((boolToByte(a1) << 1) | boolToByte(a0)) << 2
			}
			p.pixels[p.scanline*256+p.cycle] = p.palette[p.ppuRead(0x3F00+uint16(paletteIdx))%0x40]
		}

		if (p.cycle >= 2 && p.cycle < 258) || (p.cycle >= 322 && p.cycle < 338) {
			// Update shifters
			if p.PPUMASK&(1<<3) != 0 {
				p.bgShifterPatternL <<= 1
				p.bgShifterPatternH <<= 1
				p.bgShifterAttribL <<= 1
				p.bgShifterAttribH <<= 1
			}

			switch (p.cycle - 1) % 8 {
			case 0:
				// Load shifters
				p.bgShifterPatternL = (p.bgShifterPatternL & 0xFF00) | uint16(p.bgNextTileLSB)
				p.bgShifterPatternH = (p.bgShifterPatternH & 0xFF00) | uint16(p.bgNextTileMSB)
				p.bgShifterAttribL = (p.bgShifterAttribL & 0xFF00) | (uint16(p.bgNextTileAttrib) & 1) * 0xFF
				p.bgShifterAttribH = (p.bgShifterAttribH & 0xFF00) | (uint16(p.bgNextTileAttrib) >> 1) * 0xFF

				// Load tile ID
				p.bgNextTileID = p.ppuRead(0x2000 | (p.vramAddr & 0x0FFF))
			case 2:
				// Load tile attribute
				p.bgNextTileAttrib = p.ppuRead(0x23C0 | (p.vramAddr & 0x0C00) | ((p.vramAddr >> 4) & 0x38) | ((p.vramAddr >> 2) & 0x07))
				if (p.vramAddr>>1)&1 != 0 {
					p.bgNextTileAttrib >>= 4
				}
				if (p.vramAddr>>6)&1 != 0 {
					p.bgNextTileAttrib >>= 2
				}
				p.bgNextTileAttrib &= 0x03
			case 4:
				// Load tile LSB
				p.bgNextTileLSB = p.ppuRead(uint16(p.PPUCTRL&0x10)<<8 + uint16(p.bgNextTileID)<<4 + (p.vramAddr >> 12))
			case 6:
				// Load tile MSB
				p.bgNextTileMSB = p.ppuRead(uint16(p.PPUCTRL&0x10)<<8 + uint16(p.bgNextTileID)<<4 + (p.vramAddr >> 12) + 8)
			case 7:
				// Increment horizontal vram address
				if p.PPUMASK&(1<<3) != 0 {
					if p.vramAddr&0x001F == 31 {
						p.vramAddr &= ^uint16(0x001F)
						p.vramAddr ^= 0x0400
					} else {
						p.vramAddr++
					}
				}
			}
		}

		if p.cycle == 256 {
			// Increment vertical vram address
			if p.PPUMASK&(1<<3) != 0 {
				if p.vramAddr&0x7000 != 0x7000 {
					p.vramAddr += 0x1000
				} else {
					p.vramAddr &= ^uint16(0x7000)
					y := (p.vramAddr & 0x03E0) >> 5
					if y == 29 {
						y = 0
						p.vramAddr ^= 0x0800
					} else if y == 31 {
						y = 0
					} else {
						y++
					}
					p.vramAddr = (p.vramAddr & ^uint16(0x03E0)) | (y << 5)
				}
			}
		}

		if p.scanline == -1 && p.cycle >= 280 && p.cycle < 305 {
			if p.PPUMASK&(1<<3) != 0 || p.PPUMASK&(1<<4) != 0 {
				p.vramAddr = (p.vramAddr & 0x841F) | (p.vramTmpAddr & 0x7BE0)
			}
		}

		if p.cycle == 257 {
			if p.PPUMASK&(1<<3) != 0 || p.PPUMASK&(1<<4) != 0 {
				p.vramAddr = (p.vramAddr & 0xFBE0) | (p.vramTmpAddr & 0x041F)
			}
		}

		if p.scanline == 241 && p.cycle == 1 {
			p.PPUSTATUS |= 0x80
			if p.PPUCTRL&0x80 != 0 {
				p.NMI = true
			}
		}
	}

	p.cycle++
	if p.cycle >= 341 {
		p.cycle = 0
		p.scanline++
		if p.scanline >= 261 {
			p.scanline = -1
		}
	}
}
// Read reads from a PPU register.
func (p *PPU) Read(addr uint16) byte {
	switch addr {
	case 0x0002:
		status := p.PPUSTATUS
		p.PPUSTATUS &^= 0x80
		p.addrLatch = false
		return status
	case 0x0007:
		data := p.dataBuffer
		p.dataBuffer = p.ppuRead(p.vramAddr)
		if p.vramAddr >= 0x3F00 {
			data = p.dataBuffer
		}
		p.vramAddr++
		return data
	}
	return 0
}

// Write writes to a PPU register.
func (p *PPU) Write(addr uint16, data byte) {
	switch addr {
	case 0x0000:
		p.PPUCTRL = data
	case 0x0001:
		p.PPUMASK = data
	case 0x0003:
		p.OAMADDR = data
	case 0x0004:
		p.OAMDATA = data
	case 0x0005: // PPUSCROLL
		if p.addrLatch {
			p.vramTmpAddr = (p.vramTmpAddr & 0x8C1F) | ((uint16(data) & 0xF8) << 2)
			p.vramTmpAddr = (p.vramTmpAddr & 0xFBE0) | ((uint16(data) & 0x07) << 12)
			p.addrLatch = false
		} else {
			p.fineX = data & 0x07
			p.vramTmpAddr = (p.vramTmpAddr & 0xFFE0) | (uint16(data) >> 3)
			p.addrLatch = true
		}
	case 0x0006: // PPUADDR
		if p.addrLatch {
			p.vramTmpAddr = (p.vramTmpAddr & 0xFF00) | uint16(data)
			p.vramAddr = p.vramTmpAddr
			p.addrLatch = false
		} else {
			p.vramTmpAddr = (p.vramTmpAddr & 0x00FF) | ((uint16(data) & 0x3F) << 8)
			p.addrLatch = true
		}
	case 0x0007: // PPUDATA
		p.ppuWrite(p.vramAddr, data)
		if p.PPUCTRL&(1<<2) == 0 {
			p.vramAddr++
		} else {
			p.vramAddr += 32
		}
	case 0x4014:
		p.OAMDMA = data
	}
}
