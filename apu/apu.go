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

var triangleWaveform = [32]byte{
	15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
}

var noiseTimerTable = [16]uint16{
	4, 8, 16, 32, 64, 96, 128, 160, 202, 254, 380, 508, 762, 1016, 2034, 4068,
}

// PulseChannel represents a single pulse wave channel.
type PulseChannel struct {
	enabled bool

	dutyCycle        byte
	lengthCounterHalt bool // Also envelope loop flag
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

	// Envelope state
	envelopeStartFlag bool
	envelopeVolume    byte
	envelopeDivider   byte
	envelopeCounter   byte
}

// TriangleChannel represents the triangle wave channel.
type TriangleChannel struct {
	enabled bool

	lengthCounterHalt bool
	linearCounterLoad byte
	linearCounter     byte

	timer         uint16
	lengthCounter byte

	// Internal state
	timerCounter          uint16
	dutySequencer         byte
	linearCounterReloadFlag bool
}

// NoiseChannel represents the noise channel.
type NoiseChannel struct {
	enabled bool

	lengthCounterHalt bool
	constantVolume    bool
	volume            byte

	mode        bool
	timerPeriod byte

	lengthCounter byte
	shiftRegister uint16

	// Internal state
	timerCounter uint16

	// Envelope state
	envelopeStartFlag bool
	envelopeVolume    byte
	envelopeDivider   byte
	envelopeCounter   byte
}

// APU represents the Audio Processing Unit.
type APU struct {
	pulse1   *PulseChannel
	pulse2   *PulseChannel
	triangle *TriangleChannel
	noise    *NoiseChannel
	cycle    uint64

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
	apu := &APU{
		pulse1:       &PulseChannel{},
		pulse2:       &PulseChannel{},
		triangle:     &TriangleChannel{},
		noise:        &NoiseChannel{},
		sampleRate:   44100.0,
		cpuClockRate: 1789773.0,
		sampleBuffer: make([]float32, 0, 4096),
	}
	apu.noise.shiftRegister = 1
	return apu
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
	p1 := a.pulse1.output()
	p2 := a.pulse2.output()
	t := a.triangle.output()
	n := a.noise.output()

	// Approximation of NES mixing levels
	pulseOut := 0.00752 * float32(p1+p2)
	tndOut := 0.00851*float32(t) + 0.00494*float32(n) + 0.00335*float32(0) // DMC is zero for now

	return pulseOut + tndOut
}

// Clock performs one APU clock cycle.
func (a *APU) Clock() {
	// The pulse, triangle, and noise channels are clocked every CPU clock cycle.
	a.pulse1.Clock()
	a.pulse2.Clock()
	a.triangle.Clock()
	a.noise.Clock()

	// The frame counter is clocked at half the CPU speed.
	if a.cycle%2 == 0 {
		a.frameCounter++

		// 4-step sequence
		if a.sequenceMode == 0 {
			if a.frameCounter == 3729 {
				a.clockEnvelopesAndLinearCounter()
			}
			if a.frameCounter == 7457 {
				a.clockEnvelopesAndLinearCounter()
				a.clockLengthAndSweeps()
			}
			if a.frameCounter == 11186 {
				a.clockEnvelopesAndLinearCounter()
			}
			if a.frameCounter == 14915 {
				a.clockEnvelopesAndLinearCounter()
				a.clockLengthAndSweeps()
				// TODO: Fire IRQ if not inhibited
				a.frameCounter = 0
			}
		} else { // 5-step sequence
			if a.frameCounter == 3729 {
				a.clockEnvelopesAndLinearCounter()
			}
			if a.frameCounter == 7457 {
				a.clockEnvelopesAndLinearCounter()
				a.clockLengthAndSweeps()
			}
			if a.frameCounter == 11186 {
				a.clockEnvelopesAndLinearCounter()
			}
			if a.frameCounter == 18641 {
				a.clockEnvelopesAndLinearCounter()
				a.clockLengthAndSweeps()
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

func (a *APU) clockEnvelopesAndLinearCounter() {
	a.pulse1.clockEnvelope()
	a.pulse2.clockEnvelope()
	a.triangle.clockLinear()
	a.noise.clockEnvelope()
}

func (a *APU) clockLengthAndSweeps() {
	a.pulse1.clockLength()
	a.pulse1.clockSweep()
	a.pulse2.clockLength()
	a.pulse2.clockSweep()
	a.triangle.clockLength()
	a.noise.clockLength()
}

func (p *PulseChannel) clockLength() {
	if !p.lengthCounterHalt && p.lengthCounter > 0 {
		p.lengthCounter--
	}
}

func (t *TriangleChannel) clockLength() {
	if !t.lengthCounterHalt && t.lengthCounter > 0 {
		t.lengthCounter--
	}
}

func (n *NoiseChannel) clockLength() {
	if !n.lengthCounterHalt && n.lengthCounter > 0 {
		n.lengthCounter--
	}
}

func (t *TriangleChannel) clockLinear() {
	if t.linearCounterReloadFlag {
		t.linearCounter = t.linearCounterLoad
	} else if t.linearCounter > 0 {
		t.linearCounter--
	}
	if !t.lengthCounterHalt { // Control flag also halts linear counter
		t.linearCounterReloadFlag = false
	}
}


func (p *PulseChannel) clockSweep() {
	if p.sweepReloadFlag {
		p.sweepCounter = p.sweepPeriod
		p.sweepReloadFlag = false
	}

	if p.sweepCounter > 0 {
		p.sweepCounter--
	} else {
		p.sweepCounter = p.sweepPeriod
		if p.sweepEnabled && p.sweepShift > 0 {
			change := p.timer >> p.sweepShift
			if p.sweepNegate {
				p.timer -= change
				if p.timer < 8 {
					p.timer = 8
				}
			} else {
				if p.timer+change < 0x7FF {
					p.timer += change
				}
			}
		}
	}
}

func (p *PulseChannel) clockEnvelope() {
	if p.envelopeStartFlag {
		p.envelopeStartFlag = false
		p.envelopeCounter = 15
		p.envelopeDivider = p.volume
	} else if p.envelopeDivider > 0 {
		p.envelopeDivider--
	} else {
		p.envelopeDivider = p.volume
		if p.envelopeCounter > 0 {
			p.envelopeCounter--
		} else if p.lengthCounterHalt { // Loop enabled
			p.envelopeCounter = 15
		}
	}
}

func (n *NoiseChannel) clockEnvelope() {
	if n.envelopeStartFlag {
		n.envelopeStartFlag = false
		n.envelopeCounter = 15
		n.envelopeDivider = n.volume
	} else if n.envelopeDivider > 0 {
		n.envelopeDivider--
	} else {
		n.envelopeDivider = n.volume
		if n.envelopeCounter > 0 {
			n.envelopeCounter--
		} else if n.lengthCounterHalt { // Loop enabled
			n.envelopeCounter = 15
		}
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

func (t *TriangleChannel) Clock() {
	if t.timerCounter > 0 {
		t.timerCounter--
	} else {
		t.timerCounter = t.timer
		if t.linearCounter > 0 && t.lengthCounter > 0 {
			t.dutySequencer = (t.dutySequencer + 1) % 32
		}
	}
}

func (n *NoiseChannel) Clock() {
	if n.timerCounter > 0 {
		n.timerCounter--
	} else {
		n.timerCounter = noiseTimerTable[n.timerPeriod]
		
		var feedbackBit uint16
		if n.mode { // Mode 1
			feedbackBit = ((n.shiftRegister >> 6) & 1) ^ (n.shiftRegister & 1)
		} else { // Mode 0
			feedbackBit = ((n.shiftRegister >> 1) & 1) ^ (n.shiftRegister & 1)
		}
		n.shiftRegister >>= 1
		n.shiftRegister |= (feedbackBit << 14)
	}
}


// SetEnabled enables or disables the channel.
func (p *PulseChannel) SetEnabled(enabled bool) {
	p.enabled = enabled
	if !enabled {
		p.lengthCounter = 0
	}
}

func (t *TriangleChannel) SetEnabled(enabled bool) {
	t.enabled = enabled
	if !enabled {
		t.lengthCounter = 0
	}
}

func (n *NoiseChannel) SetEnabled(enabled bool) {
	n.enabled = enabled
	if !enabled {
		n.lengthCounter = 0
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
	if p.constantVolume {
		return p.volume
	}
	return p.envelopeCounter
}

func (t *TriangleChannel) output() byte {
	if !t.enabled {
		return 0
	}
	if t.lengthCounter == 0 {
		return 0
	}
	if t.linearCounter == 0 {
		return 0
	}
	if t.timer < 2 { // Supersonic
		return 0
	}
	return triangleWaveform[t.dutySequencer]
}

func (n *NoiseChannel) output() byte {
	if !n.enabled {
		return 0
	}
	if n.lengthCounter == 0 {
		return 0
	}
	// If bit 0 of the shift register is set, the volume is 0
	if (n.shiftRegister & 1) == 1 {
		return 0
	}
	if n.constantVolume {
		return n.volume
	}
	return n.envelopeCounter
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
	case addr >= 0x4004 && addr <= 0x4007:
		a.pulse2.cpuWrite(addr&0x0003, data) // Use bitwise AND for offset
	case addr >= 0x4008 && addr <= 0x400B:
		a.triangle.cpuWrite(addr, data)
	case addr >= 0x400C && addr <= 0x400F:
		a.noise.cpuWrite(addr, data)
	case addr == 0x4015: // Status register
		a.pulse1.SetEnabled(data&0x01 == 1)
		a.pulse2.SetEnabled(data&0x02 == 1)
		a.triangle.SetEnabled(data&0x04 == 1)
		a.noise.SetEnabled(data&0x08 == 1)
		// TODO: Add other channels
	case addr == 0x4017: // Frame Counter
		a.sequenceMode = (data >> 7) & 1
		a.irqInhibit = (data>>6)&1 == 1
		a.frameCounter = 0
		if a.sequenceMode == 1 {
			// 5-step mode clocks length counters and sweeps immediately
			a.clockLengthAndSweeps()
		}
	}
}

func (p *PulseChannel) cpuWrite(addr uint16, data byte) {
	switch addr {
	case 0x4000:
		p.dutyCycle = (data >> 6) & 0x03
		p.lengthCounterHalt = (data>>5)&1 == 1 // Envelope loop flag
		p.constantVolume = (data>>4)&1 == 1
		p.volume = data & 0x0F // Also envelope period
		p.envelopeStartFlag = true
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
		p.envelopeStartFlag = true
	}
}

func (t *TriangleChannel) cpuWrite(addr uint16, data byte) {
	switch addr {
	case 0x4008:
		t.lengthCounterHalt = (data>>7)&1 == 1
		t.linearCounterLoad = data & 0x7F
	case 0x4009:
		// Unused
	case 0x400A:
		t.timer = (t.timer & 0xFF00) | uint16(data)
	case 0x400B:
		t.timer = (t.timer & 0x00FF) | (uint16(data&0x07) << 8)
		t.lengthCounter = lengthCounterTable[(data>>3)&0x1F]
		t.linearCounterReloadFlag = true
	}
}

func (n *NoiseChannel) cpuWrite(addr uint16, data byte) {
	switch addr {
	case 0x400C:
		n.lengthCounterHalt = (data>>5)&1 == 1
		n.constantVolume = (data>>4)&1 == 1
		n.volume = data & 0x0F
		n.envelopeStartFlag = true
	case 0x400D:
		// Unused
	case 0x400E:
		n.mode = (data>>7)&1 == 1
		n.timerPeriod = data & 0x0F
	case 0x400F:
		n.lengthCounter = lengthCounterTable[(data>>3)&0x1F]
		n.envelopeStartFlag = true
	}
}
