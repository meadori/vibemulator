package main

import (
	"flag" // Import the flag package
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/meadori/vibemulator/bus"
	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/display"
	"github.com/meadori/vibemulator/server"
)

var (
	debugMode  = flag.Bool("debug", false, "enable debug logging")
	recordFile = flag.String("record", "", "Record gameplay to script file")
)

// logDebug prints messages if debugMode is enabled.
func logDebug(format string, a ...interface{}) {
	if *debugMode {
		log.Printf(format, a...)
	}
}

func main() {
	flag.Parse() // Parse command-line flags

	var romFilePath string
	if len(flag.Args()) > 0 {
		romFilePath = flag.Args()[0]
	}

	logDebug("Starting emulator...")
	if romFilePath != "" {
		logDebug("ROM file: %s", romFilePath)
	}

	b := bus.New()
	logDebug("Bus created.")

	if romFilePath != "" {
		cart, err := cartridge.New(romFilePath)
		if err != nil {
			log.Fatalf("Error loading ROM: %v", err)
		}
		logDebug("Cartridge loaded successfully.")

		err = b.LoadCartridge(cart)
		if err != nil {
			log.Fatalf("Error loading cartridge into bus: %v", err)
		}
		logDebug("Cartridge loaded into bus.")
	}

	// Setup recording file if requested
	var recFile *os.File
	if *recordFile != "" {
		var err error
		recFile, err = os.Create(*recordFile)
		if err != nil {
			log.Fatalf("Failed to create record file: %v", err)
		}
		defer recFile.Close()
		log.Printf("Recording gameplay to %s\n", *recordFile)
	}

	// Start the gRPC Controller Server
	grpcServer := server.NewGRPCServer()
	if err := grpcServer.Start(50051); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
	defer grpcServer.Stop()

	d := display.New(b, grpcServer, recFile)
	logDebug("Display created.")
	ebiten.SetWindowSize(display.ScaledWidth(), display.ScaledHeight())
	ebiten.SetWindowTitle("Vibemulator")
	ebiten.SetWindowResizable(true)

	logDebug("Starting Ebiten game loop...")
	if err := ebiten.RunGame(d); err != nil {
		log.Fatal(err)
	}
}
