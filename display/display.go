package display

import (
	"bytes"
	"github.com/meadori/vibemulator/cartridge"
	"github.com/sqweek/dialog"
	"image"
	"image/color"
	_ "image/png" // Required for PNG decoding
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/meadori/vibemulator/bus"
)

const (
	sampleRate       = 44100
	scalingFactor    = 1.5
	bezelWidth       = 1024
	bezelHeight      = 1024
	gameScreenX      = 318
	gameScreenY      = 322
	gameScreenWidth  = 423
	gameScreenHeight = 396
	menuBarHeight    = 20
)

type soundStream struct {
	bus *bus.Bus
}

func (s *soundStream) Read(p []byte) (n int, err error) {
	return s.bus.APU.ReadSamples(p)
}

// Display represents the emulator's display.
type Display struct {
	bus            *bus.Bus
	audioPlayer    *audio.Player
	bezelImage     *ebiten.Image
	menuBarVisible bool
}

// New creates a new Display instance.
func New(b *bus.Bus) *Display {
	audioContext := audio.NewContext(sampleRate)
	stream := &soundStream{bus: b}
	player, err := audioContext.NewPlayer(stream)
	if err != nil {
		log.Printf("Error creating audio player: %v", err)
	} else {
		player.Play()
	}

	// Load the bezel image
	bezelFile, err := os.ReadFile("display/assets/tv-bezel.png")
	if err != nil {
		log.Fatalf("Error reading bezel image: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(bezelFile))
	if err != nil {
		log.Fatalf("Error decoding bezel image: %v", err)
	}
	bezelImage := ebiten.NewImageFromImage(img)

	return &Display{
		bus:         b,
		audioPlayer: player,
		bezelImage:  bezelImage,
	}
}

func (d *Display) loadROM(path string) {
	cart, err := cartridge.New(path)
	if err != nil {
		log.Fatalf("Error loading ROM: %v", err)
	}
	d.bus.LoadCartridge(cart)
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (d *Display) Update() error {
	d.menuBarVisible = true

	// Handle menu clicks
	if d.menuBarVisible && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, _ := ebiten.CursorPosition()
		if x < 100 {
			// Load ROM
			filename, err := dialog.File().Load()
			if err != nil {
				log.Println(err)
			} else {
				d.loadROM(filename)
			}
		} else if x < 200 {
			// Exit
			os.Exit(0)
		} else if x < 300 {
			// Reset
			d.bus.Reset()
		}
	}

	// Poll controller input
	buttons := [8]bool{}
	buttons[0] = ebiten.IsKeyPressed(ebiten.KeyZ)          // A
	buttons[1] = ebiten.IsKeyPressed(ebiten.KeyX)          // B
	buttons[2] = ebiten.IsKeyPressed(ebiten.KeyShift)      // Select
	buttons[3] = ebiten.IsKeyPressed(ebiten.KeyEnter)      // Start
	buttons[4] = ebiten.IsKeyPressed(ebiten.KeyArrowUp)    // Up
	buttons[5] = ebiten.IsKeyPressed(ebiten.KeyArrowDown)  // Down
	buttons[6] = ebiten.IsKeyPressed(ebiten.KeyArrowLeft)  // Left
	buttons[7] = ebiten.IsKeyPressed(ebiten.KeyArrowRight) // Right
	d.bus.SetController1State(buttons)

	// Run the emulator for one frame's worth of PPU cycles.
	// 89342 PPU cycles per frame.
	for i := 0; i < 89342; i++ {
		d.bus.Clock()
	}

	return nil
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (d *Display) Draw(screen *ebiten.Image) {
	// Draw the bezel first, scaled
	opBezel := &ebiten.DrawImageOptions{}
	opBezel.GeoM.Scale(scalingFactor, scalingFactor)
	screen.DrawImage(d.bezelImage, opBezel)

	// Draw the game screen onto the bezel
	gameScreen := ebiten.NewImageFromImage(d.bus.PPU.GetFrame())
	opGame := &ebiten.DrawImageOptions{}

	// Scale the game screen to its target size within the bezel
	gameScaleX := float64(gameScreenWidth) / float64(gameScreen.Bounds().Dx())
	gameScaleY := float64(gameScreenHeight) / float64(gameScreen.Bounds().Dy())

	// Apply the main scaling factor to everything
	finalScaleX := gameScaleX * scalingFactor
	finalScaleY := gameScaleY * scalingFactor
	opGame.GeoM.Scale(finalScaleX, finalScaleY)

	// Apply the scaled translation
	opGame.GeoM.Translate(gameScreenX*scalingFactor, gameScreenY*scalingFactor)

	screen.DrawImage(gameScreen, opGame)

	// Draw the menu bar
	if d.menuBarVisible {
		vector.DrawFilledRect(screen, 0, 0, float32(bezelWidth*scalingFactor), menuBarHeight, color.Black, false)

		// Draw text for menu items
		// LOAD button
		loadBoxX, loadBoxY, loadBoxW, loadBoxH := float32(10), float32(4), float32(70), float32(40) // Adjusted Y
		vector.StrokeRect(screen, loadBoxX, loadBoxY, loadBoxW, loadBoxH, 1, color.White, false)
		loadText := ebiten.NewImage(50, menuBarHeight)
		ebitenutil.DebugPrintAt(loadText, "LOAD", 0, 2)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(loadBoxX+5), float64(loadBoxY+8)) // Adjusted Y-offset
		screen.DrawImage(loadText, op)

		// POWER button
		powerBoxX, powerBoxY, powerBoxW, powerBoxH := float32(90), float32(4), float32(80), float32(40) // Adjusted Y
		vector.StrokeRect(screen, powerBoxX, powerBoxY, powerBoxW, powerBoxH, 1, color.White, false)
		powerText := ebiten.NewImage(60, menuBarHeight)
		ebitenutil.DebugPrintAt(powerText, "POWER", 0, 2)
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(powerBoxX+5), float64(powerBoxY+8)) // Adjusted Y-offset
		screen.DrawImage(powerText, op)

		// RESET button
		resetBoxX, resetBoxY, resetBoxW, resetBoxH := float32(180), float32(4), float32(80), float32(40) // Adjusted Y
		vector.StrokeRect(screen, resetBoxX, resetBoxY, resetBoxW, resetBoxH, 1, color.White, false)
		resetText := ebiten.NewImage(60, menuBarHeight)
		ebitenutil.DebugPrintAt(resetText, "RESET", 0, 2)
		op = &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(resetBoxX+5), float64(resetBoxY+8)) // Adjusted Y-offset
		screen.DrawImage(resetText, op)
	}
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (d *Display) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return int(bezelWidth * scalingFactor), int(bezelHeight * scalingFactor)
}

func ScaledWidth() int {
	return int(bezelWidth * scalingFactor)
}

func ScaledHeight() int {
	return int(bezelHeight * scalingFactor)
}
