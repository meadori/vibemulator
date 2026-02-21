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
	lastButtonsP1   [8]bool
	lastButtonsP2   [8]bool
	buttonHoldCount int
	firstFrame      bool

	romLoadChan chan string

	// UI Additions
	staticImage      *ebiten.Image
	staticPix        []byte
	scanlineImage    *ebiten.Image
	currentButtons   [8]bool
	currentButtonsP2 [8]bool

	// PPU Debugger
	showDebug    bool
	debugPalette byte
	pt0Image     *ebiten.Image
	pt1Image     *ebiten.Image
	pt0Pix       []byte
	pt1Pix       []byte

	// Rewind Engine
	rewindBuffer []bus.State
	frameCount   int
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
		pt0Image:      ebiten.NewImage(128, 128),
		pt1Image:      ebiten.NewImage(128, 128),
		pt0Pix:        make([]byte, 128*128*4),
		pt1Pix:        make([]byte, 128*128*4),
		rewindBuffer:  make([]bus.State, 0, 1000), // Pre-allocate up to 1000 states (~16 seconds of rewind if sampled every frame)
	}
}

func (d *Display) loadROM(path string) {
	cart, err := cartridge.New(path)
	if err != nil {
		log.Fatalf("Error loading ROM: %v", err)
	}
	d.bus.LoadCartridge(cart)
}

func (d *Display) writeRecord(frames int, p1, p2 [8]bool) {
	formatBtns := func(b [8]bool) string {
		var names []string
		if b[0] {
			names = append(names, "A")
		}
		if b[1] {
			names = append(names, "B")
		}
		if b[2] {
			names = append(names, "SELECT")
		}
		if b[3] {
			names = append(names, "START")
		}
		if b[4] {
			names = append(names, "UP")
		}
		if b[5] {
			names = append(names, "DOWN")
		}
		if b[6] {
			names = append(names, "LEFT")
		}
		if b[7] {
			names = append(names, "RIGHT")
		}
		if len(names) == 0 {
			return "NONE"
		}
		return strings.Join(names, "+")
	}

	fmt.Fprintf(d.recordFile, "%d P1:%s P2:%s\n", frames, formatBtns(p1), formatBtns(p2))
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

	// Save States
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		log.Println("Saving State to vibemulator.sav...")
		if err := d.bus.SaveState("vibemulator.sav"); err != nil {
			log.Printf("Error saving state: %v\n", err)
		} else {
			log.Println("State saved successfully.")
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		log.Println("Loading State from vibemulator.sav...")
		if err := d.bus.LoadState("vibemulator.sav"); err != nil {
			log.Printf("Error loading state: %v\n", err)
		} else {
			log.Println("State loaded successfully.")
		}
	}

	// Debugger Toggles
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		d.showDebug = !d.showDebug
	}
	if d.showDebug && inpututil.IsKeyJustPressed(ebiten.KeyP) {
		d.debugPalette = (d.debugPalette + 1) % 8
	}

	// Rewind Engine (Prince of Persia style)
	// If holding Backspace, reverse time. Otherwise, record time.
	isRewinding := ebiten.IsKeyPressed(ebiten.KeyBackspace)

	if isRewinding && len(d.rewindBuffer) > 0 {
		// Pop the last saved state off the end of the buffer
		lastState := d.rewindBuffer[len(d.rewindBuffer)-1]
		d.rewindBuffer = d.rewindBuffer[:len(d.rewindBuffer)-1]

		// Load it instantly into the bus
		d.bus.LoadStateFromMemory(lastState)

		// We DO NOT run the emulator clock loop below, so time moves backward.
	} else if !isRewinding && d.bus.HasCartridge() {
		// Capture a snapshot every single frame for butter-smooth 1x rewind
		state := d.bus.SaveStateToMemory()
		d.rewindBuffer = append(d.rewindBuffer, state)

		// Cap the rewind buffer to 1200 states (exactly 20 seconds of 60fps gameplay history)
		if len(d.rewindBuffer) > 1200 {
			// Shift the slice left, discarding the oldest state
			copy(d.rewindBuffer, d.rewindBuffer[1:])
			d.rewindBuffer = d.rewindBuffer[:len(d.rewindBuffer)-1]
		}
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

	// Player 2
	remoteStateP2 := d.grpcServer.GetP2State()
	buttonsP2 := [8]bool{}
	buttonsP2[0] = ebiten.IsKeyPressed(ebiten.KeyI) || remoteStateP2[0] // A
	buttonsP2[1] = ebiten.IsKeyPressed(ebiten.KeyU) || remoteStateP2[1] // B
	buttonsP2[2] = ebiten.IsKeyPressed(ebiten.KeyY) || remoteStateP2[2] // Select
	buttonsP2[3] = ebiten.IsKeyPressed(ebiten.KeyH) || remoteStateP2[3] // Start
	buttonsP2[4] = ebiten.IsKeyPressed(ebiten.KeyW) || remoteStateP2[4] // Up
	buttonsP2[5] = ebiten.IsKeyPressed(ebiten.KeyS) || remoteStateP2[5] // Down
	buttonsP2[6] = ebiten.IsKeyPressed(ebiten.KeyA) || remoteStateP2[6] // Left
	buttonsP2[7] = ebiten.IsKeyPressed(ebiten.KeyD) || remoteStateP2[7] // Right
	d.bus.SetController2State(buttonsP2)
	d.currentButtonsP2 = buttonsP2

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
	if d.recordFile != nil && !isRewinding {
		if d.firstFrame {
			d.lastButtonsP1 = buttons
			d.lastButtonsP2 = buttonsP2
			d.buttonHoldCount = 1
			d.firstFrame = false
		} else {
			if buttons == d.lastButtonsP1 && buttonsP2 == d.lastButtonsP2 {
				d.buttonHoldCount++
			} else {
				d.writeRecord(d.buttonHoldCount, d.lastButtonsP1, d.lastButtonsP2)
				d.lastButtonsP1 = buttons
				d.lastButtonsP2 = buttonsP2
				d.buttonHoldCount = 1
			}
		}
	}

	// Run the emulator for one frame's worth of PPU cycles.
	// 89342 PPU cycles per frame.
	if !isRewinding {
		for i := 0; i < 89342; i++ {
			d.bus.Clock()
		}
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

	// Draw the live controller HUDs below the TV screen
	d.drawControllerHUD(screen, -160, d.currentButtons, "P1")
	d.drawControllerHUD(screen, 160, d.currentButtonsP2, "P2")

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
		logoImg := ebiten.NewImage((len(logoText)*6)+10, 16)
		ebitenutil.DebugPrintAt(logoImg, logoText, 0, 0)

		logOp := &ebiten.DrawImageOptions{}
		logOp.GeoM.Scale(3.0, 3.0)

		// Helper to draw the logo with an offset and color
		drawLogoOffset := func(dx, dy float64, c color.Color) {
			op := *logOp
			op.GeoM.Translate(350+dx, 2+dy)
			op.ColorScale.ScaleWithColor(c)
			screen.DrawImage(logoImg, &op)
		}
		// 1. Draw crisp black outline (Up, Down, Left, Right)
		black := color.RGBA{0, 0, 0, 255}
		drawLogoOffset(-1, 0, black)
		drawLogoOffset(1, 0, black)
		drawLogoOffset(0, -1, black)
		drawLogoOffset(0, 1, black)

		// 2. Draw Main Red Logo
		drawLogoOffset(0, 0, color.RGBA{220, 50, 50, 255})
	}

	// Draw PPU Debug Overlay
	if d.showDebug {
		d.drawPPUDebugOverlay(screen)
	}
}

func (d *Display) drawPPUDebugOverlay(screen *ebiten.Image) {
	// Darken background
	vector.DrawFilledRect(screen, 0, 0, float32(ScaledWidth()), float32(ScaledHeight()), color.RGBA{0, 0, 0, 220}, false)

	if !d.bus.HasCartridge() {
		ebitenutil.DebugPrintAt(screen, "LOAD A ROM TO VIEW PATTERN TABLES", ScaledWidth()/2-120, ScaledHeight()/2)
		return
	}

	// Fetch pattern tables from PPU memory without triggering IRQs
	d.bus.PPU.GetPatternTable(0, d.debugPalette, d.pt0Pix)
	d.bus.PPU.GetPatternTable(1, d.debugPalette, d.pt1Pix)
	d.pt0Image.WritePixels(d.pt0Pix)
	d.pt1Image.WritePixels(d.pt1Pix)

	// Draw tables scaled up
	scale := float64(3.0)

	op0 := &ebiten.DrawImageOptions{}
	op0.GeoM.Scale(scale, scale)
	op0.GeoM.Translate(float64(ScaledWidth())/2-(128*scale)-20, float64(ScaledHeight())/2-(64*scale))
	screen.DrawImage(d.pt0Image, op0)

	op1 := &ebiten.DrawImageOptions{}
	op1.GeoM.Scale(scale, scale)
	op1.GeoM.Translate(float64(ScaledWidth())/2+20, float64(ScaledHeight())/2-(64*scale))
	screen.DrawImage(d.pt1Image, op1)

	// Header/Footer text
	info := fmt.Sprintf("PPU PATTERN VIEWER\n\nActive Palette: %d\n[P] Cycle Palette\n[TAB] Close", d.debugPalette)
	ebitenutil.DebugPrintAt(screen, info, ScaledWidth()/2-60, 150)
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
func (d *Display) drawControllerHUD(screen *ebiten.Image, offsetX float32, activeButtons [8]bool, label string) {
	// Position the controller centered below the TV screen
	hudWidth, hudHeight := float32(300), float32(110)
	x := (float32(bezelWidth*scalingFactor)/2 - hudWidth/2) + offsetX
	y := float32(gameScreenY*scalingFactor) + float32(gameScreenHeight*scalingFactor) + 310

	// Synthwave Neon Colors
	gridColor := color.RGBA{100, 0, 200, 60}
	outerBox := color.RGBA{150, 0, 255, 150}

	cyanOn := color.RGBA{0, 255, 255, 255}
	cyanOff := color.RGBA{0, 100, 100, 255}
	yellowOn := color.RGBA{255, 255, 0, 255}
	yellowOff := color.RGBA{100, 100, 0, 255}
	magentaOn := color.RGBA{255, 0, 255, 255}
	magentaOff := color.RGBA{100, 0, 100, 255}

	// --- VIRTUAL GRID & BOX ---
	// Draw grid lines
	for i := float32(0); i <= hudWidth; i += 20 {
		vector.StrokeLine(screen, x+i, y, x+i, y+hudHeight, 1, gridColor, false)
	}
	for i := float32(0); i <= hudHeight; i += 20 {
		vector.StrokeLine(screen, x, y+i, x+hudWidth, y+i, 1, gridColor, false)
	}
	// Draw glowing outer box
	vector.StrokeRect(screen, x, y, hudWidth, hudHeight, 3, outerBox, false)
	vector.StrokeRect(screen, x+2, y+2, hudWidth-4, hudHeight-4, 1, color.RGBA{255, 255, 255, 80}, false) // Inner highlight

	// Draw Player Label
	opLbl := &ebiten.DrawImageOptions{}
	opLbl.GeoM.Scale(1.5, 1.5)
	opLbl.ColorScale.ScaleWithColor(color.RGBA{200, 200, 200, 255})
	lblImg := ebiten.NewImage(60, 16)
	ebitenutil.DebugPrintAt(lblImg, label, 0, 0)
	opP := *opLbl
	opP.GeoM.Translate(float64(x+135), float64(y-30))
	screen.DrawImage(lblImg, &opP)

	// Helper for Neon Rects
	drawNeonRect := func(rx, ry, w, h float32, active bool, onColor, offColor color.Color) {
		if active {
			vector.DrawFilledRect(screen, rx, ry, w, h, onColor, false)
			// Add a white core to the glow
			vector.DrawFilledRect(screen, rx+2, ry+2, w-4, h-4, color.RGBA{255, 255, 255, 200}, false)
		} else {
			vector.StrokeRect(screen, rx, ry, w, h, 2, offColor, false)
		}
	}

	// --- D-PAD (Cyan) ---
	dpadX, dpadY := x+60, y+55

	// Draw arms individually for a hollow cross look
	drawNeonRect(dpadX-10, dpadY-30, 20, 20, activeButtons[4], cyanOn, cyanOff) // Up
	drawNeonRect(dpadX-10, dpadY+10, 20, 20, activeButtons[5], cyanOn, cyanOff) // Down
	drawNeonRect(dpadX-30, dpadY-10, 20, 20, activeButtons[6], cyanOn, cyanOff) // Left
	drawNeonRect(dpadX+10, dpadY-10, 20, 20, activeButtons[7], cyanOn, cyanOff) // Right
	// Center square (hollow unless multiple pressed)
	vector.StrokeRect(screen, dpadX-10, dpadY-10, 20, 20, 2, cyanOff, false)

	// --- SELECT & START (Yellow) ---
	drawNeonRect(x+130, y+55, 25, 10, activeButtons[2], yellowOn, yellowOff) // Select
	drawNeonRect(x+170, y+55, 25, 10, activeButtons[3], yellowOn, yellowOff) // Start

	// --- A & B BUTTONS (Magenta) ---
	drawNeonCircle := func(cx, cy float32, active bool) {
		if active {
			vector.DrawFilledCircle(screen, cx, cy, 14, magentaOn, false)
			vector.DrawFilledCircle(screen, cx, cy, 10, color.RGBA{255, 255, 255, 200}, false) // Core
		} else {
			vector.StrokeCircle(screen, cx, cy, 14, 2, magentaOff, false)
		}
	}

	drawNeonCircle(x+230, y+60, activeButtons[1]) // B
	drawNeonCircle(x+270, y+60, activeButtons[0]) // A

	// --- NEON LABELS ---
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2.0, 2.0) // Bigger text!

	drawText := func(text string, tx, ty float64, c color.Color) {
		// Use a wider buffer so the scaled text isn't cropped
		img := ebiten.NewImage(80, 20)
		ebitenutil.DebugPrintAt(img, text, 0, 0)
		txtOp := *op
		txtOp.GeoM.Translate(tx, ty)
		txtOp.ColorScale.ScaleWithColor(c)
		screen.DrawImage(img, &txtOp)
	}

	drawText("SEL", float64(x+120), float64(y+70), yellowOff)
	drawText("STR", float64(x+160), float64(y+70), yellowOff)
	drawText("B", float64(x+223), float64(y+80), magentaOff)
	drawText("A", float64(x+263), float64(y+80), magentaOff)
}
