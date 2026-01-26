package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/meadori/vibemulator/bus"
	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/display"
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

	d := display.New(b)
	if err := ebiten.RunGame(d); err != nil {
		log.Fatal(err)
	}
}
