package main

import (
	"fmt"
	"os"

	"github.com/meadori/vibemulator/bus"
	"github.com/meadori/vibemulator/cartridge"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: vibemulator <rom_file>")
		os.Exit(1)
	}

	cart, err := cartridge.New(os.Args[1])
	if err != nil {
		fmt.Println("Error loading ROM:", err)
		os.Exit(1)
	}

	b := bus.New()
	b.LoadCartridge(cart)
	c := b.GetCPU()

	c.Reset()

	for {
		b.Clock()
	}
}
