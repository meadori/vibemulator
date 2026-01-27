package ppu

import (
	"github.com/meadori/vibemulator/cartridge" // Keep this import for Mirroring constants
	"github.com/meadori/vibemulator/mapper"     // Import mapper package
)

// PPU represents the Picture Processing Unit.
type PPU struct {
	cart *cartridge.Cartridge // Keep for general info for now, might be removed later
	mapper mapper.Mapper     // New mapper field
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

	scanlineSprites []sprite
	spriteCount     int

	spritePatternL  [8]byte // LSB of sprite pixel patterns for current scanline
	spritePatternH  [8]byte // MSB of sprite pixel patterns for current scanline
	spriteAttribute [8]byte // Attributes for sprites on current scanline
	spriteX         [8]byte // X positions for sprites on current scanline

	// Temporary storage for individual sprite pixel data
	spritePixels [8]struct {
		colorIndex byte
		priority   byte
	}
}

type sprite struct {
	y     byte
	tile  byte
	attrs byte
	x     byte
}

// New creates a new PPU instance.
func New() *PPU {
	return &PPU{
		scanlineSprites: make([]sprite, 0, 8),
	}
}

// ConnectCartridge connects a cartridge to the PPU. (Minimal use now, mainly for mapper)
func (p *PPU) ConnectCartridge(cart *cartridge.Cartridge) {
	p.cart = cart
}

// ConnectMapper connects the PPU to the mapper for CHR access and mirroring.
func (p *PPU) ConnectMapper(m mapper.Mapper) {
	p.mapper = m
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
		// CHR-RAM, delegated to mapper
		if p.mapper != nil {
			if p.mapper.PPUMapWrite(addr, data) {
				return // Handled by mapper
			}
		}
		return // Not handled by mapper (CHR-ROM is read-only)
	} else if addr >= 0x2000 && addr <= 0x3EFF {
		// Name Tables (VRAM), apply mirroring
		addr &= 0x0FFF // Maps PPU addresses $2000-$3EFF to $000-$0FFF range.
		
		// Apply mirroring logic
		var mirroredAddr uint16
		if p.mapper != nil {
			switch p.mapper.GetMirroring() {
			case cartridge.MirrorHorizontal: // Horizontal mirroring
				// $2000-$23FF, $2800-$2BFF -> VRAM $000-$3FF
				// $2400-$27FF, $2C00-$2EFF -> VRAM $400-$7FF
				mirroredAddr = addr
				if mirroredAddr >= 0x0800 { // If in NT2/NT3 range
					mirroredAddr -= 0x0800 // Map to NT0/NT1 physical space
				}
				mirroredAddr &= 0x07FF // Ensure within 2KB (0x000-0x7FF)
			case cartridge.MirrorVertical: // Vertical mirroring
				// $2000-$23FF, $2400-$27FF -> VRAM $000-$3FF (NT0/NT2)
				// $2800-$2BFF, $2C00-$2EFF -> VRAM $400-$7FF (NT1/NT3)
				mirroredAddr = addr & 0x07FF // All map to 0x000-0x7FF
			case cartridge.MirrorOneScreenLower:
				mirroredAddr = addr & 0x03FF // Map to first 1KB
			case cartridge.MirrorOneScreenUpper:
				mirroredAddr = (addr & 0x03FF) + 0x0400 // Map to second 1KB
			default:
				// Fallback, for now, use 2KB mapping without explicit mirroring logic
				mirroredAddr = addr & 0x07FF
			}
		} else {
			// No mapper, fall back to simple 2KB VRAM mapping
			mirroredAddr = addr & 0x07FF
		}
		
		p.vram[mirroredAddr] = data // Use mirroredAddr
	} else if addr >= 0x3F00 && addr <= 0x3FFF {
		// Palette RAM
		addr &= 0x001F // 32-byte palette RAM, mirrored
		if addr == 0x0010 || addr == 0x0014 || addr == 0x0018 || addr == 0x001C { // Special mirroring for background color
			addr -= 0x10 // Map to $3F00, $3F04, $3F08, $3F0C
		}
		p.palette[addr] = data
	}
}

func (p *PPU) ppuRead(addr uint16) byte {
	addr &= 0x3FFF
	if addr >= 0x0000 && addr <= 0x1FFF {
		// CHR-ROM / CHR-RAM, delegated to mapper
		if p.mapper != nil {
			data, handled := p.mapper.PPUMapRead(addr)
			if handled {
				return data
			}
		}
		return 0 // Open bus if mapper doesn't handle
	} else if addr >= 0x2000 && addr <= 0x3EFF {
		// Name Tables (VRAM), apply mirroring
		addr &= 0x0FFF // Maps PPU addresses $2000-$3EFF to $000-$0FFF range.
		
		// Apply mirroring logic (same as ppuWrite)
		var mirroredAddr uint16
		if p.mapper != nil {
			switch p.mapper.GetMirroring() {
			case cartridge.MirrorHorizontal:
				mirroredAddr = addr
				if mirroredAddr >= 0x0800 {
					mirroredAddr -= 0x0800
				}
				mirroredAddr &= 0x07FF
			case cartridge.MirrorVertical:
				mirroredAddr = addr & 0x07FF
			case cartridge.MirrorOneScreenLower:
				mirroredAddr = addr & 0x03FF
			case cartridge.MirrorOneScreenUpper:
				mirroredAddr = (addr & 0x03FF) + 0x0400
			default:
				mirroredAddr = addr & 0x07FF
			}
		} else {
			mirroredAddr = addr & 0x07FF
		}
		return p.vram[mirroredAddr]
	} else if addr >= 0x3F00 && addr <= 0x3FFF {
		// Palette RAM
		addr &= 0x001F
		if addr == 0x0010 || addr == 0x0014 || addr == 0x0018 || addr == 0x001C {
			addr -= 0x10
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
			p.PPUSTATUS &^= 0x40 // Clear sprite 0 hit
			p.PPUSTATUS &^= 0x20 // Clear sprite overflow
		}

		if p.cycle == 0 { // Start of scanline
			p.scanlineSprites = p.scanlineSprites[:0] // Clear previous scanline sprites
			p.spriteCount = 0

			// Sprite height
			spriteHeight := byte(8)
			if p.PPUCTRL&(1<<5) != 0 {
				spriteHeight = 16
			}

			// Iterate OAM to find sprites for this scanline
			oamIdx := 0
			for i := 0; i < 64; i++ { // 64 sprites in OAM
				s := sprite{
					y:     p.oam[oamIdx],
					tile:  p.oam[oamIdx+1],
					attrs: p.oam[oamIdx+2],
					x:     p.oam[oamIdx+3],
				}
				oamIdx += 4

				diff := p.scanline - int(s.y)
				if diff >= 0 && diff < int(spriteHeight) {
					if p.spriteCount < 8 {
						p.scanlineSprites = append(p.scanlineSprites, s)
						p.spriteCount++
					} else {
						p.PPUSTATUS |= 0x20 // Set sprite overflow flag
						// We still need to find sprite 0 hit, so continue iteration but don't add more sprites
					}
				}
			}
		}

		if p.scanline >= 0 && p.cycle < 256 {
			var bgPaletteIdx byte = 0
			var bgPixel byte = 0
			if p.PPUMASK&(1<<3) != 0 { // If background rendering is enabled
				mux := 0x8000 >> uint(p.fineX)
				bgPixel = (boolToByte((p.bgShifterPatternH & uint16(mux)) > 0) << 1) | boolToByte((p.bgShifterPatternL & uint16(mux)) > 0)
				bgPaletteIdx = ((boolToByte((p.bgShifterAttribH & uint16(mux)) > 0) << 1) | boolToByte((p.bgShifterAttribL & uint16(mux)) > 0)) << 2
			}

			var spritePixel byte = 0
			var spritePaletteIdx byte = 0
			var spritePriority byte = 0

			if p.PPUMASK&(1<<4) != 0 { // If sprite rendering is enabled
				for i := 0; i < p.spriteCount; i++ {
					sX := p.spriteX[i]
					if p.cycle >= int(sX) && p.cycle < int(sX+8) {
						// Calculate pixel within sprite
						offset := byte(p.cycle) - sX
						
						// Get pixel from sprite pattern data
						p1 := (p.spritePatternH[i] >> (7 - offset)) & 1
						p0 := (p.spritePatternL[i] >> (7 - offset)) & 1
						
						spritePixel = (p1 << 1) | p0
						
						if spritePixel > 0 { // If sprite pixel is opaque
							spritePaletteIdx = (p.spriteAttribute[i] & 0x03) << 2 // Get palette from attributes
							spritePriority = (p.spriteAttribute[i] >> 5) & 1    // Get priority from attributes

							if i == 0 && bgPixel > 0 && p.cycle != 255 { // Sprite 0 hit detection
								p.PPUSTATUS |= 0x40 // Set sprite 0 hit flag
							}
							break // Only render the first opaque sprite
						}
					}
				}
			}

			// Mix background and sprite pixels
			finalPixel := bgPixel
			finalPalette := bgPaletteIdx

			if bgPixel == 0 && spritePixel > 0 { // Background is transparent, sprite is opaque
				finalPixel = spritePixel
				finalPalette = spritePaletteIdx + 0x10 // Sprite palettes are 0x10-0x1F
			} else if bgPixel > 0 && spritePixel > 0 { // Both opaque
				if spritePriority == 0 { // Sprite has priority
					finalPixel = spritePixel
					finalPalette = spritePaletteIdx + 0x10
				}
				// If spritePriority == 1, background has priority, so bgPixel and bgPaletteIdx remain
			}
			p.pixels[p.scanline*256+p.cycle] = p.palette[p.ppuRead(0x3F00+uint16(finalPixel|finalPalette))%0x40]
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

		if p.cycle == 320 {
			p.fetchSpriteData()
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

	// DoOAMDMA performs Direct Memory Access from CPU memory to OAM.
func (p *PPU) DoOAMDMA(page byte, cpuRead func(uint16) byte) {
	dmaAddr := uint16(page) << 8 
	
	for i := 0; i < 256; i++ {
		p.oam[(p.OAMADDR + byte(i)) & 0xFF] = cpuRead(dmaAddr + uint16(i))
	}
}

func (p *PPU) fetchSpriteData() {
	spriteHeight := byte(8)
	if p.PPUCTRL&(1<<5) != 0 { // 8x16 sprites
		spriteHeight = 16
	}

	for i := 0; i < p.spriteCount; i++ {
		s := p.scanlineSprites[i]

		// Determine sprite pattern table address
		var patternTableAddr uint16
		if spriteHeight == 8 {
			if p.PPUCTRL&(1<<3) != 0 { // Sprite pattern table address from PPUCTRL
				patternTableAddr = 0x1000
			} else {
				patternTableAddr = 0x0000
			}
			patternTableAddr += uint16(s.tile) * 16 // Each tile is 16 bytes (8 for LSB, 8 for MSB)
		} else { // 8x16 sprites
			patternTableAddr = (uint16(s.tile&1) * 0x1000) + (uint16(s.tile&0xFE) * 16)
		}

		// Calculate row within tile
		rowInTile := byte(p.scanline) - s.y
		if s.attrs&(1<<7) != 0 { // Vertical flip
			rowInTile = spriteHeight - 1 - rowInTile
		}
		
		// Read LSB and MSB
		lsb := p.ppuRead(patternTableAddr + uint16(rowInTile))
		msb := p.ppuRead(patternTableAddr + uint16(rowInTile) + 8)

		// Horizontal flip
		if s.attrs&(1<<6) != 0 {
			lsb = reverseByte(lsb)
			msb = reverseByte(msb)
		}

		p.spritePatternL[i] = lsb
		p.spritePatternH[i] = msb
		p.spriteAttribute[i] = s.attrs
		p.spriteX[i] = s.x
	}
}

// Helper to reverse bits in a byte for horizontal flip
func reverseByte(b byte) byte {
	b = (b & 0xF0) >> 4 | (b & 0x0F) << 4
	b = (b & 0xCC) >> 2 | (b & 0x33) << 2
	b = (b & 0xAA) >> 1 | (b & 0x55) << 1
	return b
}
