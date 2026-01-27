package mapper

import (
	"fmt"

	"github.com/meadori/vibemulator/cartridge"
)

// Mapper defines the interface for different NES mappers.
type Mapper interface {
	CPUMapRead(addr uint16) (byte, bool)
	CPUMapWrite(addr uint16, data byte) bool
	PPUMapRead(addr uint16) (byte, bool)
	PPUMapWrite(addr uint16, data byte) bool
	GetMirroring() byte
}

// NewMapper creates a Mapper instance based on the cartridge's mapper ID.
func NewMapper(cart *cartridge.Cartridge) (Mapper, error) {
	switch cart.Mapper {
	case 0:
		return newNROM(cart), nil
	default:
		return nil, fmt.Errorf("unsupported mapper: %d", cart.Mapper)
	}
}
