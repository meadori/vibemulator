package cartridge

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/meadori/vibemulator/mapper"
)

// Declare logDebug function from main package
var LogDebug func(format string, a ...interface{})

// Mirroring types
const (
	MirrorHorizontal     byte = 0
	MirrorVertical       byte = 1
	MirrorOneScreenLower byte = 2
	MirrorOneScreenUpper byte = 3
	MirrorFourScreen     byte = 4
)

// Cartridge represents an NES cartridge.
type Cartridge struct {
	PRGROM   []byte
	CHRROM   []byte
	Mapper   mapper.Mapper
	Mirror   byte
	IsCHRRAM bool
}

// New creates a new Cartridge instance from a .nes file.
func New(path string) (*Cartridge, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if len(data) < 16 {
		return nil, fmt.Errorf("file is too small to be a valid NES ROM")
	}

	// Verify iNES header signature
	if data[0] != 'N' || data[1] != 'E' || data[2] != 'S' || data[3] != 0x1A {
		return nil, fmt.Errorf("invalid NES ROM format: missing iNES signature")
	}

	c := &Cartridge{}
	prgRomSize := int(data[4]) * 16384
	chrRomSize := int(data[5]) * 8192

	// Check for presence of a trainer (Bit 2 of Flag 6)
	hasTrainer := (data[6] & 0x04) != 0
	offset := 16
	if hasTrainer {
		offset += 512
	}

	// Allocate exact expected sizes to ensure compatibility even with under-dumped ROMs
	c.PRGROM = make([]byte, prgRomSize)
	if chrRomSize > 0 {
		c.CHRROM = make([]byte, chrRomSize)
		c.IsCHRRAM = false
	} else {
		c.CHRROM = make([]byte, 8192) // CHR RAM
		c.IsCHRRAM = true
	}

	// Copy PRG ROM data safely
	prgEnd := offset + prgRomSize
	if prgEnd > len(data) {
		prgEnd = len(data)
	}
	if prgEnd > offset {
		copy(c.PRGROM, data[offset:prgEnd])
	}

	// Copy CHR ROM data safely
	if chrRomSize > 0 {
		chrStart := offset + prgRomSize
		chrEnd := chrStart + chrRomSize
		if chrStart < len(data) {
			if chrEnd > len(data) {
				chrEnd = len(data)
			}
			copy(c.CHRROM, data[chrStart:chrEnd])
		}
	}

	mapperID := (data[6] >> 4) | (data[7] & 0xF0)
	c.Mirror = (data[6] & 1) | ((data[6] >> 3) & 2)

	mapper, err := NewMapper(c, mapperID)
	if err != nil {
		return nil, err
	}
	c.Mapper = mapper

	return c, nil
}

// NewMapper creates a Mapper instance based on the cartridge's mapper ID.
func NewMapper(cart *Cartridge, mapperID byte) (mapper.Mapper, error) {
	switch mapperID {
	case 0:
		return newNROM(cart), nil
	case 1:
		return newMMC1(cart), nil
	case 2:
		return newUxROM(cart), nil
	case 3:
		return newCNROM(cart), nil
	case 4:
		return newMMC3(cart), nil
	default:
		return nil, fmt.Errorf("unsupported mapper: %d", mapperID)
	}
}
