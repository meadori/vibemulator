package cartridge

import (
	"io/ioutil"
	"os"
)

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
	PRGROM []byte
	CHRROM []byte
	Mapper byte
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
	c.Mapper = (data[6] >> 4) | (data[7] & 0xF0)
	c.Mirror = (data[6] & 1) | ((data[6] >> 3) & 2)

	return c, nil
}
