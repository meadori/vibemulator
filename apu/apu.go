package apu

var lengthCounterTable = [...]byte{
	10, 254, 20, 2, 40, 4, 80, 6, 160, 8, 60, 10, 14, 12, 26, 14,
	12, 16, 24, 18, 48, 20, 96, 22, 192, 24, 72, 26, 16, 28, 32, 30,
}

var dutyCycles = [4][8]byte{
	{0, 1, 0, 0, 0, 0, 0, 0}, // 12.5%
	{0, 1, 1, 0, 0, 0, 0, 0}, // 25%
	{0, 1, 1, 1, 1, 0, 0, 0}, // 50%
	{1, 0, 0, 1, 1, 1, 1, 1}, // 25% negated
}

// PulseChannel represents a single pulse wave channel.
type PulseChannel struct {
	enabled bool

	dutyCycle        byte
	lengthCounterHalt bool
	constantVolume   bool
	volume           byte // Also used for envelope period

	sweepEnabled bool
	sweepPeriod  byte
	sweepNegate  bool
	sweepShift   byte

	timer         uint16
	lengthCounter byte

	// Internal timers and state
	timerCounter    uint16
	dutySequencer   byte
	sweepReloadFlag bool
	sweepCounter    byte
}

// APU represents the Audio Processing Unit.
type APU struct {
	pulse1 *PulseChannel
	cycle  uint64

	frameCounter      uint64
	frameSequenceStep byte
	sequenceMode      byte // 0 for 4-step, 1 for 5-step
	irqInhibit        bool

	sampleRate       float64
	cpuClockRate     float64
	sampleCycleCounter float64
	sampleBuffer     []float32
}

// New creates a new APU instance.
func New() *APU {
	return &APU{
		pulse1:       &PulseChannel{},
		sampleRate:   44100.0,
		cpuClockRate: 1789773.0,
		sampleBuffer: make([]float32, 0, 4096),
	}
}

// ReadSamples reads generated samples into a byte buffer.
func (a *APU) ReadSamples(p []byte) (n int, err error) {
	numSamples := len(p) / 4 // 2 channels, 2 bytes each
	if numSamples > len(a.sampleBuffer) {
		numSamples = len(a.sampleBuffer)
	}

	written := 0
	for i := 0; i < numSamples; i++ {
		sample := a.sampleBuffer[i]
		sample16 := int16(sample * 32767)
		p[written] = byte(sample16)
		p[written+1] = byte(sample16 >> 8)
		p[written+2] = byte(sample16)
		p[written+3] = byte(sample16 >> 8)
		written += 4
	}

	// Drain the buffer
	a.sampleBuffer = a.sampleBuffer[numSamples:]

	return written, nil
}


// output returns the current mixed audio sample.
func (a *APU) output() float32 {
	// For now, just output the first pulse channel.
	return float32(a.pulse1.output()) / 15.0
}

// Clock performs one APU clock cycle.
func (a *APU) Clock() {
	// The pulse channels are clocked every CPU clock cycle.
	a.pulse1.Clock()

	// The frame counter is clocked at half the CPU speed.
	if a.cycle%2 == 0 {
		a.frameCounter++

		// 4-step sequence
		if a.sequenceMode == 0 {
			if a.frameCounter == 3729 {
				// Step 1: Clock envelopes and sweeps
			}
			if a.frameCounter == 7457 {
				// Step 2: Clock envelopes, sweeps, and length counters
				a.clockLengthCountersAndSweeps()
			}
			if a.frameCounter == 11186 {
				// Step 3: Clock envelopes and sweeps
			}
			if a.frameCounter == 14915 {
				// Step 4: Clock envelopes, sweeps, and length counters
				a.clockLengthCountersAndSweeps()
				// TODO: Fire IRQ if not inhibited
				a.frameCounter = 0
			}
		} else { // 5-step sequence
			if a.frameCounter == 3729 {
				// Step 1: Clock envelopes and sweeps
			}
			if a.frameCounter == 7457 {
				// Step 2: Clock envelopes, sweeps, and length counters
				a.clockLengthCountersAndSweeps()
			}
			if a.frameCounter == 11186 {
				// Step 3: Clock envelopes and sweeps
			}
			if a.frameCounter == 18641 {
				// Step 5: Clock envelopes, sweeps, and length counters
				a.clockLengthCountersAndSweeps()
				a.frameCounter = 0
			}
		}
	}

	// Downsample to the desired sample rate.
	a.sampleCycleCounter += a.sampleRate / a.cpuClockRate
	if a.sampleCycleCounter >= 1 {
		a.sampleCycleCounter--
		a.sampleBuffer = append(a.sampleBuffer, a.output())
	}


	a.cycle++
}

func (a *APU) clockLengthCountersAndSweeps() {
	a.pulse1.clockLength()
	// TODO: Add other channels
}

func (p *PulseChannel) clockLength() {
	if !p.lengthCounterHalt && p.lengthCounter > 0 {
		p.lengthCounter--
	}
}

func (p *PulseChannel) Clock() {
	if p.timerCounter > 0 {
		p.timerCounter--
	} else {
		p.timerCounter = p.timer
		p.dutySequencer = (p.dutySequencer + 1) % 8
	}
}

// SetEnabled enables or disables the channel.
func (p *PulseChannel) SetEnabled(enabled bool) {
	p.enabled = enabled
	if !enabled {
		p.lengthCounter = 0
	}
}

func (p *PulseChannel) output() byte {
	if !p.enabled {
		return 0
	}
	if p.lengthCounter == 0 {
		return 0
	}
	if p.timer < 8 { // Supersonic frequencies
		return 0
	}
	if dutyCycles[p.dutyCycle][p.dutySequencer] == 0 {
		return 0
	}
	// TODO: Add envelope and sweep logic
	return p.volume
}

// CPURead handles CPU reads from the APU's registers.
func (a *APU) CPURead(addr uint16) byte {
	var data byte
	// Register read logic will go here.
	return data
}

// CPUWrite handles CPU writes to the APU's registers.
func (a *APU) CPUWrite(addr uint16, data byte) {
	switch {
	case addr >= 0x4000 && addr <= 0x4003:
		a.pulse1.cpuWrite(addr, data)
	case addr == 0x4015: // Status register
		a.pulse1.SetEnabled(data&0x01 == 1)
		// TODO: Add other channels
	case addr == 0x4017: // Frame Counter
		a.sequenceMode = (data >> 7) & 1
		a.irqInhibit = (data>>6)&1 == 1
		a.frameCounter = 0
		if a.sequenceMode == 1 {
			// 5-step mode clocks length counters and sweeps immediately
			a.clockLengthCountersAndSweeps()
		}
	}
}

func (p *PulseChannel) cpuWrite(addr uint16, data byte) {
	switch addr {
	case 0x4000:
		p.dutyCycle = (data >> 6) & 0x03
		p.lengthCounterHalt = (data>>5)&1 == 1
		p.constantVolume = (data>>4)&1 == 1
		p.volume = data & 0x0F
	case 0x4001:
		p.sweepEnabled = (data>>7)&1 == 1
		p.sweepPeriod = (data >> 4) & 0x07
		p.sweepNegate = (data>>3)&1 == 1
		p.sweepShift = data & 0x07
		p.sweepReloadFlag = true
	case 0x4002:
		p.timer = (p.timer & 0xFF00) | uint16(data)
	case 0x4003:
		p.timer = (p.timer & 0x00FF) | (uint16(data&0x07) << 8)
		p.lengthCounter = lengthCounterTable[(data>>3)&0x1F]
		p.dutySequencer = 0 // Reset phase
	}
}
