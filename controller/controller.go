package controller

// Controller represents a standard NES controller.
type Controller struct {
	buttons [8]bool // A, B, Select, Start, Up, Down, Left, Right
	index   byte    // The current bit being read from the shift register
	strobe  byte    // The strobe latch
}

// New creates a new Controller instance.
func New() *Controller {
	return &Controller{}
}

// SetButtons updates the state of the controller's buttons.
func (c *Controller) SetButtons(buttons [8]bool) {
	c.buttons = buttons
}

// Write handles CPU writes to the controller register ($4016 or $4017).
func (c *Controller) Write(data byte) {
	c.strobe = data & 1
	if c.strobe == 1 {
		c.index = 0 // Strobe high, reset the read index
	}
}

// Read handles CPU reads from the controller register.
func (c *Controller) Read() byte {
	if c.index >= 8 {
		return 1 // After the 8 main buttons, standard controllers return 1.
	}

	value := byte(0)
	if c.buttons[c.index] {
		value = 1
	}

	// If strobe is low, the shift register is advanced on each read.
	if c.strobe == 0 {
		c.index++
	}

	return value
}
