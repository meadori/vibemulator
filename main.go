package main

import (
	"fmt"

	"github.com/meadori/vibemulator/bus"
)

func main() {
	b := bus.New()
	c := b.GetCPU()

	// Load program
	b.Write(0x8000, 0xA9)
	b.Write(0x8001, 0x42)

	// Set reset vector
	b.Write(0xFFFC, 0x00)
	b.Write(0xFFFD, 0x80)

	c.Reset()

	// Run for a few cycles
	for i := 0; i < 10; i++ {
		c.Clock()
	}

	fmt.Printf("Accumulator: 0x%X\n", c.A)
}
