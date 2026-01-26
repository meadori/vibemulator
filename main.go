package main

import (
	"fmt"

	"github.com/meadori/vibemulator/bus"
)

func main() {
	b := bus.New()
	b.Write(0x0100, 0x42)
	fmt.Printf("Read from RAM: 0x%X\n", b.Read(0x0100))
}
