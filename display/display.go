package display

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/meadori/vibemulator/bus"
)

const (
	sampleRate = 44100
)

type soundStream struct {
	bus *bus.Bus
}

func (s *soundStream) Read(p []byte) (n int, err error) {
	return s.bus.APU.ReadSamples(p)
}

// Display represents the emulator's display.
type Display struct {
	bus         *bus.Bus
	audioPlayer *audio.Player
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

	return &Display{bus: b, audioPlayer: player}
}

// Update proceeds the game state.
// Update is called every tick (1/60 [s] by default).
func (d *Display) Update() error {
	// Poll controller input
	buttons := [8]bool{}
	buttons[0] = ebiten.IsKeyPressed(ebiten.KeyZ)         // A
	buttons[1] = ebiten.IsKeyPressed(ebiten.KeyX)         // B
	buttons[2] = ebiten.IsKeyPressed(ebiten.KeyShift)     // Select
	buttons[3] = ebiten.IsKeyPressed(ebiten.KeyEnter)     // Start
	buttons[4] = ebiten.IsKeyPressed(ebiten.KeyArrowUp)   // Up
	buttons[5] = ebiten.IsKeyPressed(ebiten.KeyArrowDown) // Down
	buttons[6] = ebiten.IsKeyPressed(ebiten.KeyArrowLeft) // Left
	buttons[7] = ebiten.IsKeyPressed(ebiten.KeyArrowRight)// Right
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
	screen.WritePixels(d.bus.PPU.GetFrame().Pix)
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (d *Display) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 256, 240
}