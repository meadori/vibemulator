package main

import (
	"flag" // Import the flag package
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/meadori/vibemulator/bus"
	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/display"
)

var (
	debugMode = flag.Bool("debug", false, "enable debug logging")
)

// logDebug prints messages if debugMode is enabled.
func logDebug(format string, a ...interface{}) {
	if *debugMode {
		log.Printf(format, a...)
	}
}

func main() {
	flag.Parse() // Parse command-line flags

	if len(flag.Args()) != 1 { // flag.Args() returns non-flag arguments
		fmt.Println("Usage: vibemulator [-debug] <rom_file>")
		os.Exit(1)
	}
	romFilePath := flag.Args()[0]

	logDebug("Starting emulator...")
	logDebug("ROM file: %s", romFilePath)

	cart, err := cartridge.New(romFilePath)
	if err != nil {
		log.Fatalf("Error loading ROM: %v", err)
	}
	logDebug("Cartridge loaded successfully.")

	b := bus.New()
	logDebug("Bus created.")
	err = b.LoadCartridge(cart)
	if err != nil {
		log.Fatalf("Error loading cartridge into bus: %v", err)
	}
	logDebug("Cartridge loaded into bus.")

	d := display.New(b)
	logDebug("Display created.")
	logDebug("Starting Ebiten game loop...")
	if err := ebiten.RunGame(d); err != nil {
		log.Fatal(err)
	}
}
