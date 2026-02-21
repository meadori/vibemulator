package ppu

import "image/color"

// DebugMapper allows side-effect free reads for debug views.
type DebugMapper interface {
	PPUDebugRead(addr uint16) (byte, bool)
}

// PPUDebugRead safely reads PPU memory without triggering hardware side effects (like MMC3's A12 counter).
func (p *PPU) PPUDebugRead(addr uint16) byte {
	var data byte
	addr &= 0x3FFF

	switch {
	case addr <= 0x1FFF:
		if p.cart != nil {
			if dm, ok := p.cart.Mapper.(DebugMapper); ok {
				data, _ = dm.PPUDebugRead(addr)
			} else {
				data, _ = p.cart.Mapper.PPUMapRead(addr)
			}
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

// GetPatternTable extracts the requested pattern table (0 or 1) into a 128x128 RGBA byte slice using the specified palette index (0-7).
func (p *PPU) GetPatternTable(i int, palette byte, dest []byte) {
	for tileY := 0; tileY < 16; tileY++ {
		for tileX := 0; tileX < 16; tileX++ {
			offset := uint16(tileY*256 + tileX*16)
			for row := uint16(0); row < 8; row++ {
				tileLSB := p.PPUDebugRead(uint16(i)*0x1000 + offset + row)
				tileMSB := p.PPUDebugRead(uint16(i)*0x1000 + offset + row + 8)

				for col := 0; col < 8; col++ {
					pixel := (tileLSB & 0x01) | ((tileMSB & 0x01) << 1)
					tileLSB >>= 1
					tileMSB >>= 1

					// Decode from right to left
					x := tileX*8 + (7 - col)
					y := tileY*8 + int(row)

					colorIndex := p.PPUDebugRead(0x3F00 + uint16(palette)*4 + uint16(pixel))
					var c color.RGBA
					if pixel == 0 {
						// Render background color as black/transparent for the debugger overlay instead of actual BG color for clarity
						c = color.RGBA{0, 0, 0, 255}
					} else {
						c = p.SystemPalette[colorIndex]
					}

					idx := (y*128 + x) * 4
					dest[idx] = c.R
					dest[idx+1] = c.G
					dest[idx+2] = c.B
					dest[idx+3] = 255
				}
			}
		}
	}
}
