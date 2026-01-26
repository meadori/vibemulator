package main

import (
	"fmt"

	"github.com/meadori/vibemulator/bus"
)

func main() {
	b := bus.New()
	fmt.Printf("Hello, vibemulator! Bus: %+v\n", b)
}
