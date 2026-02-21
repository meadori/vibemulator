package display

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png" // Required for PNG decoding
	"log"
	"math/rand"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/sqweek/dialog"

	"github.com/meadori/vibemulator/bus"
	"github.com/meadori/vibemulator/cartridge"
	"github.com/meadori/vibemulator/server"
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
	grpcServer      *server.GRPCServer

	// Recording fields
	recordFile      *os.File
	lastButtons     [8]bool
	buttonHoldCount int
	firstFrame      bool

	romLoadChan chan string

	// UI Additions
	staticImage    *ebiten.Image
	staticPix      []byte
	scanlineImage  *ebiten.Image
	currentButtons [8]bool
}

// New creates a new Display instance.
func New(b *bus.Bus, srv *server.GRPCServer, recFile *os.File) *Display {
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

	// Create TV Static assets
	staticImg := ebiten.NewImage(256, 240)
	staticPix := make([]byte, 256*240*4)

	// Create CRT Scanlines overlay (black line every other row)
	scanImg := ebiten.NewImage(256, 240)
	for y := 0; y < 240; y += 2 {
		vector.DrawFilledRect(scanImg, 0, float32(y), 256, 1, color.RGBA{0, 0, 0, 70}, false)
	}

	return &Display{
		bus:           b,
		audioPlayer:   player,
		bezelImage:    bezelImage,
		grpcServer:    srv,
		recordFile:    recFile,
		firstFrame:    true,
		romLoadChan:   make(chan string, 1),
		staticImage:   staticImg,
		staticPix:     staticPix,
		scanlineImage: scanImg,
	}
}

func (d *Display) loadROM(path string) {
	cart, err := cartridge.New(path)
	if err != nil {
		log.Fatalf("Error loading ROM: %v", err)
	}
	d.bus.LoadCartridge(cart)
}

func (d *Display) writeRecord(frames int, b [8]bool) {
	var btnNames []string
	if b[0] {
		btnNames = append(btnNames, "A")
	}
	if b[1] {
		btnNames = append(btnNames, "B")
	}
	if b[2] {
		btnNames = append(btnNames, "SELECT")
	}
	if b[3] {
		btnNames = append(btnNames, "START")
	}
	if b[4] {
		btnNames = append(btnNames, "UP")
	}
	if b[5] {
		btnNames = append(btnNames, "DOWN")
	}
	if b[6] {
		btnNames = append(btnNames, "LEFT")
	}
	if b[7] {
		btnNames = append(btnNames, "RIGHT")
	}

	btnStr := "NONE"
	if len(btnNames) > 0 {
		btnStr = strings.Join(btnNames, "+")
	}
	fmt.Fprintf(d.recordFile, "%d %s\n", frames, btnStr)
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (d *Display) Update() error {
	d.menuBarVisible = true

	// Check if a ROM was selected via the async dialog
	select {
	case filename := <-d.romLoadChan:
		d.loadROM(filename)
	default:
	}

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
				go func() {
					filename, err := dialog.File().Load()
					if err != nil {
						log.Println(err)
					} else {
						d.romLoadChan <- filename
					}
				}()
			}
		}
	}

	if d.resetBlinkTimer > 0 {
		d.resetBlinkTimer--
	}

	// Poll controller input (Logical OR local input and remote network input)
	remoteState := d.grpcServer.GetP1State()
	buttons := [8]bool{}
	buttons[0] = ebiten.IsKeyPressed(ebiten.KeyZ) || remoteState[0]          // A
	buttons[1] = ebiten.IsKeyPressed(ebiten.KeyX) || remoteState[1]          // B
	buttons[2] = ebiten.IsKeyPressed(ebiten.KeyShift) || remoteState[2]      // Select
	buttons[3] = ebiten.IsKeyPressed(ebiten.KeyEnter) || remoteState[3]      // Start
	buttons[4] = ebiten.IsKeyPressed(ebiten.KeyArrowUp) || remoteState[4]    // Up
	buttons[5] = ebiten.IsKeyPressed(ebiten.KeyArrowDown) || remoteState[5]  // Down
	buttons[6] = ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || remoteState[6]  // Left
	buttons[7] = ebiten.IsKeyPressed(ebiten.KeyArrowRight) || remoteState[7] // Right
	d.bus.SetController1State(buttons)
	d.currentButtons = buttons

	// Generate TV Static if no cartridge is loaded
	if !d.bus.HasCartridge() {
		for i := 0; i < len(d.staticPix); i += 4 {
			val := byte(rand.Intn(256))
			d.staticPix[i] = val
			d.staticPix[i+1] = val
			d.staticPix[i+2] = val
			d.staticPix[i+3] = 255
		}
		d.staticImage.WritePixels(d.staticPix)
	}

	// Record inputs if recording is enabled
	if d.recordFile != nil {
		if d.firstFrame {
			d.lastButtons = buttons
			d.buttonHoldCount = 1
			d.firstFrame = false
		} else {
			if buttons == d.lastButtons {
				d.buttonHoldCount++
			} else {
				d.writeRecord(d.buttonHoldCount, d.lastButtons)
				d.lastButtons = buttons
				d.buttonHoldCount = 1
			}
		}
	}

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

	// Determine what to show on the TV
	var rawScreen *ebiten.Image
	if d.bus.HasCartridge() {
		rawScreen = ebiten.NewImageFromImage(d.bus.PPU.GetFrame())
		// Apply CRT Scanlines directly over the game frame before scaling
		rawScreen.DrawImage(d.scanlineImage, nil)
	} else {
		rawScreen = d.staticImage
	}

	// Scale the game screen to its target size within the bezel
	gameScaleX := float64(gameScreenWidth) / float64(rawScreen.Bounds().Dx())
	gameScaleY := float64(gameScreenHeight) / float64(rawScreen.Bounds().Dy())

	opGame := &ebiten.DrawImageOptions{}
	// Apply the main scaling factor to everything
	finalScaleX := gameScaleX * scalingFactor
	finalScaleY := gameScaleY * scalingFactor
	opGame.GeoM.Scale(finalScaleX, finalScaleY)

	// Apply the scaled translation
	opGame.GeoM.Translate(gameScreenX*scalingFactor, gameScreenY*scalingFactor)

	screen.DrawImage(rawScreen, opGame)

	// Draw the live controller HUD below the TV screen
	d.drawControllerHUD(screen)

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

// drawControllerHUD draws a live NES controller below the TV screen that lights up when buttons are pressed.
func (d *Display) drawControllerHUD(screen *ebiten.Image) {
	// Position the controller centered below the TV screen
	hudWidth, hudHeight := float32(300), float32(110)
	x := float32(bezelWidth*scalingFactor)/2 - hudWidth/2
	y := float32(gameScreenY*scalingFactor) + float32(gameScreenHeight*scalingFactor) + 60

	// Base shell (light grey with dark stripe)
	vector.DrawFilledRect(screen, x, y, hudWidth, hudHeight, color.RGBA{180, 180, 180, 255}, false)
	vector.DrawFilledRect(screen, x+20, y+hudHeight/2-10, hudWidth-40, 20, color.RGBA{30, 30, 30, 255}, false)

	// D-Pad (Up=4, Down=5, Left=6, Right=7)
	dpadX, dpadY := x+55, y+55
	dpadColor := color.RGBA{20, 20, 20, 255}
	hlColor := color.RGBA{130, 130, 130, 255}

	// Draw cross
	vector.DrawFilledRect(screen, dpadX-12, dpadY-35, 24, 70, dpadColor, false) // Vert
	vector.DrawFilledRect(screen, dpadX-35, dpadY-12, 70, 24, dpadColor, false) // Horiz

	// D-Pad Highlights
	if d.currentButtons[4] { // Up
		vector.DrawFilledRect(screen, dpadX-12, dpadY-35, 24, 25, hlColor, false)
	}
	if d.currentButtons[5] { // Down
		vector.DrawFilledRect(screen, dpadX-12, dpadY+10, 24, 25, hlColor, false)
	}
	if d.currentButtons[6] { // Left
		vector.DrawFilledRect(screen, dpadX-35, dpadY-12, 25, 24, hlColor, false)
	}
	if d.currentButtons[7] { // Right
		vector.DrawFilledRect(screen, dpadX+10, dpadY-12, 25, 24, hlColor, false)
	}

	// Select/Start (Select=2, Start=3)
	selColor, startColor := color.RGBA{30, 30, 30, 255}, color.RGBA{30, 30, 30, 255}
	if d.currentButtons[2] {
		selColor = hlColor
	}
	if d.currentButtons[3] {
		startColor = hlColor
	}
	// Angled pills (simulated with rectangles for now)
	vector.DrawFilledRect(screen, x+120, y+60, 35, 12, selColor, false)
	vector.DrawFilledRect(screen, x+170, y+60, 35, 12, startColor, false)

	// B/A Buttons (B=1, A=0)
	bColor, aColor := color.RGBA{200, 0, 0, 255}, color.RGBA{200, 0, 0, 255}
	btnHlColor := color.RGBA{255, 100, 100, 255}
	if d.currentButtons[1] {
		bColor = btnHlColor
	}
	if d.currentButtons[0] {
		aColor = btnHlColor
	}
	// A is higher than B on NES
	vector.DrawFilledCircle(screen, x+230, y+70, 18, bColor, false)
	vector.DrawFilledCircle(screen, x+275, y+60, 18, aColor, false)
}
