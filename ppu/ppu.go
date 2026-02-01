package ppu

import (
	"image"
	"image/color"

	"github.com/meadori/vibemulator/cartridge"
)

// Declare logDebug function from main package
var LogDebug func(format string, a ...interface{})

// PPU represents the Picture Processing Unit.
type PPU struct {
	cart         *cartridge.Cartridge
	nt_map       [4]uint16
	vram         [2048]byte
	oam          [256]byte
	palette      [32]byte
	Scanline     int
	Cycle        int
	Status       byte
	Mask         byte
	Ctrl         byte
	vramAddr     uint16
	vramTmpAddr  uint16
	fineX        byte
	addrLatch    byte
	ppuData      byte
	oamAddr      byte
	FrameCounter int
	NMI          bool

	// Frame buffer
	frame *image.RGBA

	// System Palette
	SystemPalette [0x40]color.RGBA

	// Shifters
	bgPatternShifterLo uint16
	bgPatternShifterHi uint16
	bgAttribShifterLo  uint16
	bgAttribShifterHi  uint16
	bgNextTileID       byte
	bgNextTileAttrib   byte
	bgNextTileLSB      byte
	bgNextTileMSB      byte

	// Sprite rendering
	spriteScanline []spriteInfo
	spriteZeroHit  bool
	spriteZero     bool
	spriteEvalCycle int
	sprite0InScanline bool
	spriteCount       byte
}

type spriteInfo struct {
	y, id, attr, x byte
}

// Reset resets the PPU state.
func (p *PPU) Reset() {
	p.Scanline = 0
	p.Cycle = 0
	p.Status = 0x00
	p.Mask = 0x00
	p.Ctrl = 0x00
	p.vramAddr = 0x0000
	p.vramTmpAddr = 0x0000
	p.fineX = 0x00
	p.addrLatch = 0x00
	p.ppuData = 0x00
	p.oamAddr = 0x00
	p.FrameCounter = 0
	p.NMI = false

	p.spriteEvalCycle = 0
	p.sprite0InScanline = false
	p.spriteScanline = p.spriteScanline[:0] // Clear the secondary OAM

	p.spriteCount = 0

	p.bgPatternShifterLo = 0x0000
	p.bgPatternShifterHi = 0x0000
	p.bgAttribShifterLo = 0x0000
	p.bgAttribShifterHi = 0x0000
	p.bgNextTileID = 0x00
	p.bgNextTileAttrib = 0x00
	p.bgNextTileLSB = 0x00
	p.bgNextTileMSB = 0x00
	p.loadBGShifters()

	// Initialize palette RAM to 0x0F (black) on reset
	for i := range p.palette {
		p.palette[i] = 0x0F
	}
}

// New creates a new PPU instance.
func New() *PPU {
	p := &PPU{
		frame: image.NewRGBA(image.Rect(0, 0, 256, 240)),
	}
	p.SystemPalette = getSystemPalette()

	p.spriteScanline = make([]spriteInfo, 8)
	p.Reset() // Call Reset here to initialize state
	return p
}

// GetFrame returns the current frame.
func (p *PPU) GetFrame() *image.RGBA {
	return p.frame
}

// ConnectCartridge connects the cartridge to the PPU.
func (p *PPU) ConnectCartridge(cart *cartridge.Cartridge) {
	p.cart = cart
	mirror := p.cart.Mapper.GetMirroring()
	if mirror == cartridge.MirrorVertical {
		p.nt_map = [4]uint16{0x0000, 0x0400, 0x0000, 0x0400}
	} else if mirror == cartridge.MirrorHorizontal {
		p.nt_map = [4]uint16{0x0000, 0x0000, 0x0400, 0x0400}
	}
}

// Clock performs one PPU clock cycle.
func (p *PPU) Clock() {
	if p.cart == nil {
		return
	}
	renderingEnabled := (p.Mask & 0x08) != 0 || (p.Mask & 0x10) != 0 // Check if background or sprites are enabled

	if p.Scanline == -1 && p.Cycle == 339 && renderingEnabled && p.FrameCounter%2 == 1 {
		// On odd frames, last cycle of pre-render scanline (339, 1-indexed) is skipped if rendering is enabled.
		// This means we immediately advance to the next scanline/frame without processing cycle 340.
		p.Cycle = 0
		p.Scanline = 0 // Wrap to scanline 0, cycle 0
		p.FrameCounter++
		return // Skip rest of Clock() function for this "skipped" cycle
	}
	// --- END NEW LOGIC ---

	if p.Scanline >= -1 && p.Scanline < 240 {
		// Original incorrect odd frame cycle skip removed.

		if p.Scanline == -1 && p.Cycle == 1 {
			p.Status &= 0x1F
			p.spriteZeroHit = false
		}

		if (p.Cycle >= 1 && p.Cycle < 258) || (p.Cycle >= 322 && p.Cycle < 338) {
			p.updateShifters()

			switch (p.Cycle - 1) % 8 {
			case 0:
				p.bgNextTileID = p.PPURead(0x2000 | (p.vramAddr & 0x0FFF))
			case 2:
				p.bgNextTileAttrib = p.PPURead(0x23C0 | (p.vramAddr & 0x0C00) | ((p.vramAddr >> 4) & 0x38) | ((p.vramAddr >> 2) & 0x07))
				if (p.vramAddr & 0x0040) != 0 {
					p.bgNextTileAttrib >>= 4
				}
				if (p.vramAddr & 0x0002) != 0 {
					p.bgNextTileAttrib >>= 2
				}
				p.bgNextTileAttrib &= 0x03
			case 4:
				p.bgNextTileLSB = p.PPURead((uint16(p.Ctrl>>4)&1)*0x1000 + uint16(p.bgNextTileID)*16 + ((p.vramAddr >> 12) & 0x07))
			case 6:
				p.bgNextTileMSB = p.PPURead((uint16(p.Ctrl>>4)&1)*0x1000 + uint16(p.bgNextTileID)*16 + ((p.vramAddr >> 12) & 0x07) + 8)
			case 7:
				p.incrementScrollX()
				p.loadBGShifters()
			}
		}
		if p.Cycle == 256 {
			p.incrementScrollY()
		}

		if p.Cycle == 257 {
			p.transferAddressX()
		}

		// Sprite evaluation initialization (occurs at cycle 257 for all renderable scanlines)
		if p.Cycle == 257 && p.Scanline >= -1 && p.Scanline < 240 {
			// Clear secondary OAM (p.spriteScanline)
			p.spriteScanline = p.spriteScanline[:0]
			p.spriteCount = 0
			p.sprite0InScanline = false
			p.oamAddr = 0 // OAMADDR is set to 0 at dot 257 of each scanline if rendering is enabled.
			p.Status &= 0xDF // Clear Sprite Overflow flag ($2002 bit 5)
		}

		// Cycle-accurate sprite evaluation will be implemented here.

		if p.Cycle >= 257 && p.Cycle <= 320 && p.Scanline >= -1 && p.Scanline < 240 {
			oamIndex := (p.Cycle - 257) * 4 // current sprite in OAM to evaluate (0 to 63)
			if oamIndex < 256 { // Ensure we don't go out of bounds for OAM (256 bytes)
				y := p.oam[oamIndex]
				id := p.oam[oamIndex+1]
				attr := p.oam[oamIndex+2]
				x := p.oam[oamIndex+3]

				spriteHeight := byte(8)
				if (p.Ctrl & 0x08) != 0 { // PPUCTRL bit 5 for 8x16 sprites
					spriteHeight = 16
				}

				// Check if sprite is visible on the *next* scanline (p.Scanline + 1)
				// The +1 is because sprite Y coordinate is top-most scanline of sprite - 1
				if (p.Scanline+1) >= int(y) && (p.Scanline+1) < int(y)+int(spriteHeight) {
					if p.spriteCount < 8 {
						p.spriteScanline = append(p.spriteScanline, spriteInfo{
							y:    y,
							id:   id,
							attr: attr,
							x:    x,
						})
						if oamIndex == 0 { // Check if sprite 0 is found (first entry in primary OAM)
							p.sprite0InScanline = true
						}
					}
					// Increment spriteCount regardless of whether it was added to spriteScanline
					p.spriteCount++
					if p.spriteCount > 8 { // Set Sprite Overflow flag immediately if 9th sprite is found
						p.Status |= 0x20
					}
				}
			}
		}





		if p.Scanline == -1 && p.Cycle >= 280 && p.Cycle < 305 {
			p.transferAddressY()
		}
	}

	if p.Scanline == 241 && p.Cycle == 1 {
		p.Status |= 0x80
		if (p.Ctrl & 0x80) != 0 {
			p.NMI = true
		}
	}

	if p.Scanline < 240 && p.Cycle >= 1 && p.Cycle <= 256 {
		p.renderPixel()
	}

	p.Cycle++
	if p.Cycle > 340 {
		p.Cycle = 0
		p.Scanline++
		if p.Scanline > 260 {
			p.Scanline = -1
			p.FrameCounter++
		}
	}
}

// PPURead reads from PPU memory.
func (p *PPU) PPURead(addr uint16) byte {
	var data byte
	addr &= 0x3FFF

	switch {
	case addr <= 0x1FFF:
		if p.cart != nil {
			data, _ = p.cart.Mapper.PPUMapRead(addr)
		}
	case addr >= 0x2000 && addr <= 0x3EFF:
		addr &= 0x0FFF
		data = p.vram[p.getMirrorAddress(addr)]
	case addr >= 0x3F00 && addr <= 0x3FFF:
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
		data = p.palette[addr]
	}

	return data
}

// PPUWrite writes to PPU memory.
func (p *PPU) PPUWrite(addr uint16, data byte) {
	addr &= 0x3FFF

	switch {
	case addr <= 0x1FFF:
		if p.cart != nil {
			p.cart.Mapper.PPUMapWrite(addr, data)
		}
	case addr >= 0x2000 && addr <= 0x3EFF:
		addr &= 0x0FFF
		p.vram[p.getMirrorAddress(addr)] = data
	case addr >= 0x3F00 && addr <= 0x3FFF:
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

func (p *PPU) getMirrorAddress(addr uint16) uint16 {
	nametableIndex := (addr >> 10) & 3
	offset := addr & 0x03FF
	return p.nt_map[nametableIndex] + offset
}

// CPURead reads from PPU registers.
func (p *PPU) CPURead(addr uint16) byte {
	var data byte
	switch addr {
	case 0x0000: // Control
	case 0x0001: // Mask
	case 0x0002: // Status
		data = (p.Status & 0xE0) | (p.ppuData & 0x1F)
		if p.spriteZeroHit {
			data |= 0x40
		}
		p.Status &= 0x7F // Clear VBlank flag
		p.addrLatch = 0
	case 0x0003: // OAM Address
	case 0x0004: // OAM Data
		data = p.oam[p.oamAddr]
	case 0x0005: // Scroll
	case 0x0006: // PPU Address
	case 0x0007: // PPU Data
		data = p.ppuData // Always return the buffered data

		// Update the buffer with the content at current vramAddr
		// If vramAddr is in the palette range, fetch from the mirrored nametable for the buffer
		if p.vramAddr >= 0x3F00 {
			p.ppuData = p.PPURead(p.vramAddr - 0x1000) // Read from mirrored nametable space for buffer
			data = p.PPURead(p.vramAddr)               // Return actual palette value immediately
		} else {
			p.ppuData = p.PPURead(p.vramAddr)
		}

		if (p.Ctrl & 0x04) != 0 {
			p.vramAddr += 32
		} else {
			p.vramAddr++
		}
	}
	return data
}

// CPUWrite writes to PPU registers.
func (p *PPU) CPUWrite(addr uint16, data byte) {
	switch addr {
	case 0x0000: // Control
		p.Ctrl = data
		p.vramTmpAddr = (p.vramTmpAddr & 0xF3FF) | ((uint16(data) & 0x03) << 10)
	case 0x0001: // Mask
		p.Mask = data
	case 0x0002: // Status
	case 0x0003: // OAM Address
		p.oamAddr = data
	case 0x0004: // OAM Data
		p.oam[p.oamAddr] = data
		p.oamAddr++
	case 0x0005: // Scroll
		if p.addrLatch == 0 {
			p.fineX = data & 0x07
			p.vramTmpAddr = (p.vramTmpAddr & 0xFFE0) | (uint16(data) >> 3)
			p.addrLatch = 1
		} else {
			p.vramTmpAddr = (p.vramTmpAddr & 0x8C1F) | ((uint16(data) & 0x07) << 12) | ((uint16(data) & 0xF8) << 2)
			p.addrLatch = 0
		}
	case 0x0006: // PPU Address
		if p.addrLatch == 0 {
			p.vramTmpAddr = (p.vramTmpAddr & 0x00FF) | ((uint16(data) & 0x3F) << 8)
			p.addrLatch = 1
		} else {
			p.vramTmpAddr = (p.vramTmpAddr & 0xFF00) | uint16(data)
			p.vramAddr = p.vramTmpAddr
			p.addrLatch = 0
		}
	case 0x0007: // PPU Data
		p.PPUWrite(p.vramAddr, data)
		if (p.Ctrl & 0x04) != 0 {
			p.vramAddr += 32
		} else {
			p.vramAddr++
		}
	}
}

// DoOAMDMA performs OAM DMA transfer.
func (p *PPU) DoOAMDMA(data [256]byte) {
	for i := 0; i < 256; i++ {
		p.oam[byte((uint16(p.oamAddr) + uint16(i)) % 256)] = data[i]
	}
}

func (p *PPU) loadBGShifters() {
	p.bgPatternShifterLo = (p.bgPatternShifterLo & 0x00FF) | (uint16(p.bgNextTileLSB) << 8)
	p.bgPatternShifterHi = (p.bgPatternShifterHi & 0x00FF) | (uint16(p.bgNextTileMSB) << 8)

	if (p.bgNextTileAttrib & 0x01) != 0 {
		p.bgAttribShifterLo = (p.bgAttribShifterLo & 0x00FF) | 0xFF00
	} else {
		p.bgAttribShifterLo = (p.bgAttribShifterLo & 0x00FF) | 0x0000
	}
	if (p.bgNextTileAttrib & 0x02) != 0 {
		p.bgAttribShifterHi = (p.bgAttribShifterHi & 0x00FF) | 0xFF00
	} else {
		p.bgAttribShifterHi = (p.bgAttribShifterHi & 0x00FF) | 0x0000
	}
}

func (p *PPU) updateShifters() {
	if (p.Mask & 0x08) != 0 {
		p.bgPatternShifterLo <<= 1
		p.bgPatternShifterHi <<= 1
		p.bgAttribShifterLo <<= 1
		p.bgAttribShifterHi <<= 1
	}
}

// Scrolling
func (p *PPU) incrementScrollX() {
	if (p.Mask & 0x08) != 0 {
		if (p.vramAddr & 0x001F) == 31 {
			p.vramAddr &= 0xFFE0
			p.vramAddr ^= 0x0400
		} else {
			p.vramAddr++
		}
	}
}

func (p *PPU) incrementScrollY() {
	if (p.Mask & 0x08) != 0 {
		if (p.vramAddr & 0x7000) != 0x7000 { // if fine Y < 7
			p.vramAddr += 0x1000 // increment fine Y
		} else {
			p.vramAddr &= 0x8FFF // fine Y = 0
			y := (p.vramAddr & 0x03E0) >> 5
			if y == 29 {
				y = 0
				p.vramAddr ^= 0x0800 // switch nametable Y
			} else if y == 31 {
				y = 0
			} else {
				y++
			}
			p.vramAddr = (p.vramAddr & 0xFC1F) | (y << 5)
		}
	}
}

func (p *PPU) transferAddressX() {
	if (p.Mask & 0x08) != 0 {
		p.vramAddr = (p.vramAddr & 0xFBE0) | (p.vramTmpAddr & 0x041F)
	}
}

func (p *PPU) transferAddressY() {
	if (p.Mask & 0x08) != 0 {
		p.vramAddr = (p.vramAddr & 0x841F) | (p.vramTmpAddr & 0x7BE0)
	}
}



func (p *PPU) renderPixel() {

	var bgPixel byte

	var bgPalette byte

	var mux uint16 // Declared outside the if block

	var p1, p2, a1, a2 bool // Declared outside the if block



	if (p.Mask & 0x08) != 0 {

		mux = 0x8000 >> p.fineX

		p1 = (p.bgPatternShifterLo & uint16(mux)) > 0

		p2 = (p.bgPatternShifterHi & uint16(mux)) > 0

		bgPixel = (boolToByte(p2) << 1) | boolToByte(p1)



		a1 = (p.bgAttribShifterLo & uint16(mux)) > 0

		a2 = (p.bgAttribShifterHi & uint16(mux)) > 0

		bgPalette = (boolToByte(a2) << 1) | boolToByte(a1)

	}

	var spPixel byte
	var spPalette byte
	var spPriority bool
	if (p.Mask & 0x10) != 0 {
		// p.spriteZero = false // This flag was for tracking if *this* pixel is sprite 0. Removed.
		for i := 0; i < len(p.spriteScanline); i++ {
			if p.Cycle-1 >= int(p.spriteScanline[i].x) && p.Cycle-1 < int(p.spriteScanline[i].x)+8 {
				// No longer setting p.spriteZero here. p.sprite0InScanline is set during evaluation.
				spritePatternAddrLo := uint16(p.Ctrl&0x20)*0x1000 + uint16(p.spriteScanline[i].id)*16 + (uint16(p.Scanline) - uint16(p.spriteScanline[i].y))
				if p.spriteScanline[i].attr&0x80 != 0 {
					spritePatternAddrLo = uint16(p.Ctrl&0x20)*0x1000 + uint16(p.spriteScanline[i].id)*16 + (7 - (uint16(p.Scanline) - uint16(p.spriteScanline[i].y)))
				}
				spritePatternAddrHi := spritePatternAddrLo + 8

				var spritePatternBitLo byte
				var spritePatternBitHi byte
				if p.spriteScanline[i].attr&0x40 != 0 { // horizontal flip
					shift := byte(p.Cycle - 1 - int(p.spriteScanline[i].x))
					spritePatternBitLo = (p.PPURead(spritePatternAddrLo) >> shift) & 0x01
					spritePatternBitHi = (p.PPURead(spritePatternAddrHi) >> shift) & 0x01
				} else {
					shift := byte(7 - (p.Cycle - 1 - int(p.spriteScanline[i].x)))
					spritePatternBitLo = (p.PPURead(spritePatternAddrLo) >> shift) & 0x01
					spritePatternBitHi = (p.PPURead(spritePatternAddrHi) >> shift) & 0x01
				}

				spPixel = (spritePatternBitHi << 1) | spritePatternBitLo
				spPalette = (p.spriteScanline[i].attr & 0x03) + 0x04
				spPriority = (p.spriteScanline[i].attr & 0x20) == 0

				if spPixel != 0 {
					break
				}
			}
		}
	}

	var finalPixel byte
	var finalPalette byte

	if bgPixel != 0 && spPixel != 0 {
		if spPriority {
			finalPixel = spPixel
			finalPalette = spPalette
		} else {
			finalPixel = bgPixel
			finalPalette = bgPalette
		}
		if p.sprite0InScanline && p.spriteZeroHit == false && p.Cycle < 255 {
			p.spriteZeroHit = true
		}
	} else if bgPixel == 0 && spPixel != 0 {
		finalPixel = spPixel
		finalPalette = spPalette
	} else if bgPixel != 0 && spPixel == 0 {
		finalPixel = bgPixel
		finalPalette = bgPalette
	} else {
		finalPixel = 0
		finalPalette = 0
	}

	colorIndex := p.PPURead(0x3F00 + uint16(finalPalette)*4 + uint16(finalPixel))
	p.frame.Set(p.Cycle-1, p.Scanline, p.SystemPalette[colorIndex])
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func getSystemPalette() [0x40]color.RGBA {
	return [0x40]color.RGBA{
		{84, 84, 84, 255}, {0, 30, 116, 255}, {8, 16, 144, 255}, {48, 0, 136, 255}, {68, 0, 100, 255}, {92, 0, 48, 255}, {84, 4, 0, 255}, {60, 24, 0, 255}, {32, 42, 0, 255}, {8, 58, 0, 255}, {0, 64, 0, 255}, {0, 60, 0, 255}, {0, 50, 60, 255}, {0, 0, 0, 255}, {0, 0, 0, 255}, {0, 0, 0, 255},
		{152, 150, 152, 255}, {8, 76, 196, 255}, {48, 50, 236, 255}, {92, 30, 228, 255}, {136, 20, 176, 255}, {160, 20, 100, 255}, {152, 34, 32, 255}, {120, 60, 0, 255}, {84, 90, 0, 255}, {40, 114, 0, 255}, {8, 124, 0, 255}, {0, 118, 40, 255}, {0, 102, 120, 255}, {0, 0, 0, 255}, {0, 0, 0, 255}, {0, 0, 0, 255},
		{236, 238, 236, 255}, {76, 154, 236, 255}, {120, 124, 236, 255}, {176, 98, 236, 255}, {228, 84, 236, 255}, {236, 88, 180, 255}, {236, 106, 100, 255}, {212, 136, 32, 255}, {160, 170, 0, 255}, {116, 196, 0, 255}, {76, 208, 32, 255}, {56, 204, 108, 255}, {56, 180, 204, 255}, {60, 60, 60, 255}, {0, 0, 0, 255}, {0, 0, 0, 255},
		{236, 238, 236, 255}, {168, 204, 236, 255}, {188, 188, 236, 255}, {212, 178, 236, 255}, {236, 174, 236, 255}, {236, 174, 212, 255}, {236, 180, 176, 255}, {228, 196, 144, 255}, {204, 210, 120, 255}, {180, 222, 120, 255}, {168, 226, 144, 255}, {152, 226, 180, 255}, {160, 214, 228, 255}, {160, 162, 160, 255}, {0, 0, 0, 255}, {0, 0, 0, 255},
	}
}