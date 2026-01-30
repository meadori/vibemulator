package cartridge

import (
	"fmt"
	"io/ioutil"
	"os"
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

// Mapper defines the interface for different NES mappers.
type Mapper interface {
	CPUMapRead(addr uint16) (byte, bool)
	CPUMapWrite(addr uint16, data byte) bool
	PPUMapRead(addr uint16) (byte, bool)
	PPUMapWrite(addr uint16, data byte) bool
	GetMirroring() byte
}

// Cartridge represents an NES cartridge.
type Cartridge struct {
	PRGROM []byte
	CHRROM []byte
	Mapper Mapper
	Mirror byte
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

	c := &Cartridge{}
	prgRomSize := int(data[4]) * 16384
	chrRomSize := int(data[5]) * 8192

	c.PRGROM = data[16 : 16+prgRomSize]
	
	if chrRomSize > 0 {
		c.CHRROM = data[16+prgRomSize : 16+prgRomSize+chrRomSize]
	} else {
		// Allocate CHR RAM if no CHR ROM is present
		c.CHRROM = make([]byte, 8192) // Common size for CHR RAM
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
func NewMapper(cart *Cartridge, mapperID byte) (Mapper, error) {
	switch mapperID {
	case 0:
		return newNROM(cart), nil
	default:
		return nil, fmt.Errorf("unsupported mapper: %d", mapperID)
	}
}