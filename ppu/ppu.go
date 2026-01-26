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

	scanline int
	cycle int

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

func (p *PPU) ppuWrite(addr uint16, data byte) {
	addr &= 0x3FFF
	if addr >= 0x0000 && addr <= 0x1FFF {
		p.cart.CHRROM[addr] = data
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
			p.pixels[p.scanline*256+p.cycle] = p.palette[p.ppuRead(0x3F00)%0x40]
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
	case 0x0005:
		if p.addrLatch {
			p.vramTmpAddr = (p.vramTmpAddr & 0xFF00) | uint16(data)
			p.vramAddr = p.vramTmpAddr
		} else {
			p.vramTmpAddr = (p.vramTmpAddr & 0x00FF) | (uint16(data) << 8)
		}
		p.addrLatch = !p.addrLatch
	case 0x0006:
		if p.addrLatch {
			p.vramTmpAddr = (p.vramTmpAddr & 0xFF00) | uint16(data)
			p.vramAddr = p.vramTmpAddr
		} else {
			p.vramTmpAddr = (p.vramTmpAddr & 0x00FF) | (uint16(data) << 8)
		}
		p.addrLatch = !p.addrLatch
	case 0x0007:
		p.ppuWrite(p.vramAddr, data)
		p.vramAddr++
	case 0x4014:
		p.OAMDMA = data
	}
}
