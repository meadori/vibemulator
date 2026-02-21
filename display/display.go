package display

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png" // Required for PNG decoding
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sqweek/dialog"

	"github.com/meadori/vibemulator/bus"
	"github.com/meadori/vibemulator/cartridge"
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
	menuBarHeight    = 50
)

type soundStream struct {
	bus *bus.Bus
}

func (s *soundStream) Read(p []byte) (n int, err error) {
	return s.bus.APU.ReadSamples(p)
}

// Display represents the emulator's display.
type Display struct {
	bus             *bus.Bus
	audioPlayer     *audio.Player
	bezelImage      *ebiten.Image
	menuBarVisible  bool
	resetBlinkTimer int
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
	if d.menuBarVisible && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cx, cy := ebiten.CursorPosition()
		x, y := float32(cx), float32(cy)

		if y >= 5 && y <= 45 { // Inside the button Y boundaries
			if x >= 60 && x <= 140 {
				// POWER (Exit)
				os.Exit(0)
			} else if x >= 150 && x <= 230 {
				// RESET
				d.bus.Reset()
				d.resetBlinkTimer = 30 // Blink for half a second (30 frames)
			} else if x >= 240 && x <= 320 {
				// LOAD
				filename, err := dialog.File().Load()
				if err != nil {
					log.Println(err)
				} else {
					d.loadROM(filename)
				}
			}
		}
	}

	if d.resetBlinkTimer > 0 {
		d.resetBlinkTimer--
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
		// Draw a light-grey chassis color for the top bar
		vector.DrawFilledRect(screen, 0, 0, float32(bezelWidth*scalingFactor), menuBarHeight, color.RGBA{190, 190, 190, 255}, false)
		// Draw a dark stripe below it to separate the menu from the TV
		vector.DrawFilledRect(screen, 0, menuBarHeight, float32(bezelWidth*scalingFactor), 4, color.RGBA{40, 40, 40, 255}, false)

		cx, cy := ebiten.CursorPosition()
		mouseX, mouseY := float32(cx), float32(cy)
		isMouseDown := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

		// Power LED area (X: 10 to 50)
		ledX := float32(30)
		ledY := float32(25)
		// Recessed dark square for the LED
		vector.DrawFilledRect(screen, ledX-10, ledY-10, 20, 20, color.RGBA{30, 30, 30, 255}, false)

		// Blink logic: If timer is active, toggle glow every 4 frames. If timer is 0, stay solidly glowing.
		if d.resetBlinkTimer == 0 || (d.resetBlinkTimer/4)%2 == 0 {
			// LED glow (outer)
			vector.DrawFilledCircle(screen, ledX, ledY, 8, color.RGBA{200, 0, 0, 80}, false)
			// LED glow (inner)
			vector.DrawFilledCircle(screen, ledX, ledY, 5, color.RGBA{255, 0, 0, 180}, false)
			// LED core
			vector.DrawFilledCircle(screen, ledX, ledY, 3, color.RGBA{255, 100, 100, 255}, false)
		} else {
			// LED off (dark core)
			vector.DrawFilledCircle(screen, ledX, ledY, 3, color.RGBA{100, 0, 0, 255}, false)
		}

		// POWER button (X: 60 to 140)
		powerHover := mouseX >= 60 && mouseX <= 140 && mouseY >= 5 && mouseY <= 45
		drawNESButton(screen, "POWER", 60, 5, 80, 40, powerHover, powerHover && isMouseDown)

		// RESET button (X: 150 to 230)
		resetHover := mouseX >= 150 && mouseX <= 230 && mouseY >= 5 && mouseY <= 45
		drawNESButton(screen, "RESET", 150, 5, 80, 40, resetHover, resetHover && isMouseDown)

		// LOAD button (X: 240 to 320)
		loadHover := mouseX >= 240 && mouseX <= 320 && mouseY >= 5 && mouseY <= 45
		drawNESButton(screen, "LOAD", 240, 5, 80, 40, loadHover, loadHover && isMouseDown)

		// VIBEMULATOR Logo (X: 350+)
		logoText := "VIBEMULATOR"
		logoImg := ebiten.NewImage(len(logoText)*6, 16)
		ebitenutil.DebugPrintAt(logoImg, logoText, 0, 0)

		logOp := &ebiten.DrawImageOptions{}
		logOp.GeoM.Scale(2.5, 2.5)
		logOp.GeoM.Skew(-0.15, 0)
		logOp.GeoM.Translate(350, 4)
		// NES Red logo
		logOp.ColorScale.ScaleWithColor(color.RGBA{220, 50, 50, 255})
		screen.DrawImage(logoImg, logOp)
	}
}

func drawNESButton(screen *ebiten.Image, textStr string, x, y, w, h float32, isHovered, isPressed bool) {
	// Classic NES grey plastic button colors - lightened slightly
	baseColor := color.RGBA{70, 70, 70, 255}
	lightColor := color.RGBA{120, 120, 120, 255}
	darkColor := color.RGBA{40, 40, 40, 255}

	if isHovered {
		baseColor = color.RGBA{85, 85, 85, 255}
		lightColor = color.RGBA{140, 140, 140, 255}
	}

	if isPressed {
		// Swap highlight and shadow to invert the bevel and create a "pushed in" effect
		lightColor, darkColor = darkColor, lightColor
	}

	// Draw main background
	vector.DrawFilledRect(screen, x, y, w, h, baseColor, false)

	// Draw 3D Bevels
	borderSize := float32(4)

	// Top border
	vector.DrawFilledRect(screen, x, y, w, borderSize, lightColor, false)
	// Left border
	vector.DrawFilledRect(screen, x, y, borderSize, h, lightColor, false)
	// Bottom border
	vector.DrawFilledRect(screen, x, y+h-borderSize, w, borderSize, darkColor, false)
	// Right border
	vector.DrawFilledRect(screen, x+w-borderSize, y, borderSize, h, darkColor, false)

	// Draw Text
	textImg := ebiten.NewImage(len(textStr)*6, 16)
	ebitenutil.DebugPrintAt(textImg, textStr, 0, 0)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)

	// Center the text in the button
	textW := float32(len(textStr) * 6 * 2)
	textH := float32(16 * 2)
	textX := x + (w-textW)/2
	textY := y + (h-textH)/2 + 4 // slight downward offset for debug font

	if isPressed {
		textX += 2
		textY += 2
	}

	op.GeoM.Translate(float64(textX), float64(textY))
	// NES Red text
	op.ColorScale.ScaleWithColor(color.RGBA{220, 50, 50, 255})
	screen.DrawImage(textImg, op)
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
