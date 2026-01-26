package cartridge

import (
	"io/ioutil"
	"os"
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
	c.PRGROM = data[16 : 16+int(data[4])*16384]
	c.CHRROM = data[16+int(data[4])*16384:]
	c.Mapper = (data[6] >> 4) | (data[7] & 0xF0)
	c.Mirror = (data[6] & 1) | ((data[6] >> 3) & 2)

	return c, nil
}
