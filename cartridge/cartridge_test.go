package cartridge

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	// Create a dummy .nes file
	header := []byte{0x4E, 0x45, 0x53, 0x1A, 0x02, 0x01, 0x31, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	prg := make([]byte, 2*16384)
	chr := make([]byte, 1*8192)
	data := append(header, prg...)
	data = append(data, chr...)

	tmpfile, err := ioutil.TempFile("", "test.nes")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(data); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cart, err := New(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if len(cart.PRGROM) != 2*16384 {
		t.Errorf("Expected PRGROM size to be %d, but got %d", 2*16384, len(cart.PRGROM))
	}
	if len(cart.CHRROM) != 1*8192 {
		t.Errorf("Expected CHRROM size to be %d, but got %d", 1*8192, len(cart.CHRROM))
	}
	if cart.Mapper != 3 {
		t.Errorf("Expected mapper to be 3, but got %d", cart.Mapper)
	}
}
