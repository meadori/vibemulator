package display

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/meadori/vibemulator/bus"
)

// Declare logDebug function from main package
var LogDebug func(format string, a ...interface{})

// Display represents the emulator's display.
type Display struct {
	bus *bus.Bus
}

// New creates a new Display instance.
func New(b *bus.Bus) *Display {
	return &Display{bus: b}
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (d *Display) Update() error {
	// The NES CPU runs at 1.79 MHz, and the PPU runs at 3x that speed.
	// For a 60 FPS frame rate, we have approximately 29,780 PPU cycles per frame.
	for i := 0; i < 29780; i++ {
		d.bus.Clock()
	}
	return nil
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (d *Display) Draw(screen *ebiten.Image) {
	screen.WritePixels(d.bus.PPU.GetFrame().Pix)
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (d *Display) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 256, 240
}