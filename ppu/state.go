package ppu

type State struct {
	Nt_map                                                                                                                            [4]uint16
	Vram                                                                                                                              [2048]byte
	Oam                                                                                                                               [256]byte
	Palette                                                                                                                           [32]byte
	Scanline, Cycle, FrameCounter, SpriteEvalCycle                                                                                    int
	Status, Mask, Ctrl, FineX, AddrLatch, PpuData, OamAddr, BgNextTileID, BgNextTileAttrib, BgNextTileLSB, BgNextTileMSB, SpriteCount byte
	VramAddr, VramTmpAddr, BgPatternShifterLo, BgPatternShifterHi, BgAttribShifterLo, BgAttribShifterHi                               uint16
	NMI, SpriteZeroHit, SpriteZero, Sprite0InScanline                                                                                 bool
	FrameBuffer                                                                                                                       []byte
}

func (p *PPU) SaveState() State {
	fb := make([]byte, len(p.frame.Pix))
	copy(fb, p.frame.Pix)

	return State{
		p.nt_map, p.vram, p.oam, p.palette, p.Scanline, p.Cycle, p.FrameCounter, p.spriteEvalCycle,
		p.Status, p.Mask, p.Ctrl, p.fineX, p.addrLatch, p.ppuData, p.oamAddr, p.bgNextTileID, p.bgNextTileAttrib, p.bgNextTileLSB, p.bgNextTileMSB, p.spriteCount,
		p.vramAddr, p.vramTmpAddr, p.bgPatternShifterLo, p.bgPatternShifterHi, p.bgAttribShifterLo, p.bgAttribShifterHi,
		p.NMI, p.spriteZeroHit, p.spriteZero, p.sprite0InScanline,
		fb,
	}
}

func (p *PPU) LoadState(s State) {
	p.nt_map, p.vram, p.oam, p.palette, p.Scanline, p.Cycle, p.FrameCounter, p.spriteEvalCycle = s.Nt_map, s.Vram, s.Oam, s.Palette, s.Scanline, s.Cycle, s.FrameCounter, s.SpriteEvalCycle
	p.Status, p.Mask, p.Ctrl, p.fineX, p.addrLatch, p.ppuData, p.oamAddr, p.bgNextTileID, p.bgNextTileAttrib, p.bgNextTileLSB, p.bgNextTileMSB, p.spriteCount = s.Status, s.Mask, s.Ctrl, s.FineX, s.AddrLatch, s.PpuData, s.OamAddr, s.BgNextTileID, s.BgNextTileAttrib, s.BgNextTileLSB, s.BgNextTileMSB, s.SpriteCount
	p.vramAddr, p.vramTmpAddr, p.bgPatternShifterLo, p.bgPatternShifterHi, p.bgAttribShifterLo, p.bgAttribShifterHi = s.VramAddr, s.VramTmpAddr, s.BgPatternShifterLo, s.BgPatternShifterHi, s.BgAttribShifterLo, s.BgAttribShifterHi
	p.NMI, p.spriteZeroHit, p.spriteZero, p.sprite0InScanline = s.NMI, s.SpriteZeroHit, s.SpriteZero, s.Sprite0InScanline

	if len(s.FrameBuffer) == len(p.frame.Pix) {
		copy(p.frame.Pix, s.FrameBuffer)
	}
}
