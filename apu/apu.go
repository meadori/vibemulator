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

var dmcRateTable = [16]uint16{
	428, 380, 340, 320, 286, 254, 226, 214, 190, 160, 142, 128, 106, 84, 72, 54,
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

// DMCChannel represents the delta modulation channel.
type DMCChannel struct {
	enabled bool

	irqEnabled bool
	loop       bool
	rateIndex  byte
	
	timer uint16
	
	// Sample state
	sampleAddress  uint16
	sampleLength   uint16
	currentAddress uint16
	bytesRemaining uint16
	
	// Output state
	outputLevel     byte
	shiftRegister   byte
	bitsRemaining   byte
	sampleBuffer    byte
	sampleBufferEmpty bool

	irqPending bool // New field to signal IRQ
	bus BusReader // Interface to read from the bus
}

// APU represents the Audio Processing Unit.
type APU struct {
	pulse1   *PulseChannel
	pulse2   *PulseChannel
	triangle *TriangleChannel
	noise    *NoiseChannel
	dmc      *DMCChannel
	cycle    uint64
	bus      BusReader // Interface to read from the bus

	frameCounter      uint64
	frameSequenceStep byte
	sequenceMode      byte // 0 for 4-step, 1 for 5-step
	irqInhibit        bool
	DmcIRQ bool // DMC Interrupt Flag

	sampleRate       float64
	cpuClockRate     float64
	sampleCycleCounter float64
	sampleBuffer     []float32
}

// BusReader defines the interface the APU needs to read from the bus.
type BusReader interface {
	Read(addr uint16) byte
}


// New creates a new APU instance.
func New() *APU {
	apu := &APU{
		pulse1:       &PulseChannel{},
		pulse2:       &PulseChannel{},
		triangle:     &TriangleChannel{},
		noise:        &NoiseChannel{},
		dmc:          &DMCChannel{},
		sampleRate:   44100.0,
		cpuClockRate: 1789773.0,
		sampleBuffer: make([]float32, 0, int(44100*2)), // Increased capacity for 2 seconds of audio
	}
	apu.noise.shiftRegister = 1
	return apu
}

// ConnectBus connects the bus to the APU.
func (a *APU) ConnectBus(bus BusReader) {
	a.bus = bus
	a.dmc.ConnectBus(bus)
}

// ConnectBus connects the bus to the DMCChannel.
func (d *DMCChannel) ConnectBus(bus BusReader) {
	d.bus = bus
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
	d := a.dmc.output()

	// Approximation of NES mixing levels
	pulseOut := 0.00752 * float32(p1+p2)
	tndOut := 0.00851*float32(t) + 0.00494*float32(n) + 0.00335*float32(d)

	return pulseOut + tndOut
}

// Clock performs one APU clock cycle.
func (a *APU) Clock() {
	// The pulse, triangle, and noise channels are clocked every CPU clock cycle.
	a.pulse1.Clock()
	a.pulse2.Clock()
	a.triangle.Clock()
	a.noise.Clock()
	a.dmc.Clock(a.bus)

	// Check for DMC IRQ
	    if a.dmc.irqPending {
	        a.DmcIRQ = true
	    }
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

func (d *DMCChannel) Clock(bus BusReader) {
	if d.timer > 0 {
		d.timer--
	} else {
		d.timer = dmcRateTable[d.rateIndex]
		if d.bitsRemaining == 0 {
			d.bitsRemaining = 8
			if d.sampleBufferEmpty && d.bytesRemaining > 0 {
				d.sampleBuffer = bus.Read(d.currentAddress)
				d.sampleBufferEmpty = false
				d.currentAddress++
				if d.currentAddress == 0 {
					d.currentAddress = 0x8000
				}
				d.bytesRemaining--
				if d.bytesRemaining == 0 {
					if d.loop {
						d.currentAddress = d.sampleAddress
						d.bytesRemaining = d.sampleLength
					} else { // Sample finished, generate IRQ if enabled
						if d.irqEnabled {
							d.irqPending = true // Set IRQ pending flag
						}
					}
				}
			}
		}

		if !d.sampleBufferEmpty {
			if (d.shiftRegister & 1) == 1 {
				if d.outputLevel <= 125 {
					d.outputLevel += 2
				}
			} else {
				if d.outputLevel >= 2 {
					d.outputLevel -= 2
				}
			}
			d.shiftRegister >>= 1
			d.bitsRemaining--
			if d.bitsRemaining == 0 {
				d.sampleBufferEmpty = true
			}
		}
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

func (d *DMCChannel) SetEnabled(enabled bool) {
	d.enabled = enabled
	if !enabled {
		d.bytesRemaining = 0
	} else {
		if d.bytesRemaining == 0 {
			d.currentAddress = d.sampleAddress
			d.bytesRemaining = d.sampleLength
		}
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

func (d *DMCChannel) output() byte {
	return d.outputLevel
}

// CPURead handles CPU reads from the APU's registers.
func (a *APU) CPURead(addr uint16) byte {
	var data byte
	switch addr {
	case 0x4015: // Status register
		// Bits 0-4: Length counter status for Pulse 1, Pulse 2, Triangle, Noise, DMC
		if a.pulse1.lengthCounter > 0 {
			data |= 0x01
		}
		if a.pulse2.lengthCounter > 0 {
			data |= 0x02
		}
		if a.triangle.lengthCounter > 0 {
			data |= 0x04
		}
		if a.noise.lengthCounter > 0 {
			data |= 0x08
		}
		if a.dmc.bytesRemaining > 0 { // DMC status is bytes remaining, not length counter
			data |= 0x10
		}
		// Bit 6: Frame Interrupt Flag (cleared on read)
		// Bit 7: DMC Interrupt Flag (cleared on read)
        if a.DmcIRQ {
            data |= 0x80
            a.DmcIRQ = false
            a.dmc.irqPending = false
        }
        // Frame Interrupt Flag (bit 6) is cleared on read only if not inhibited
        if !a.irqInhibit {
            // TODO: Need a frame IRQ flag in APU struct
            // For now, if we had a frame IRQ, we would clear it here.
        }

	}
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
	case addr >= 0x4010 && addr <= 0x4013:
		a.dmc.cpuWrite(addr, data)
	case addr == 0x4015: // Status register
		a.pulse1.SetEnabled(data&0x01 == 1)
		a.pulse2.SetEnabled(data&0x02 == 1)
		a.triangle.SetEnabled(data&0x04 == 1)
		a.noise.SetEnabled(data&0x08 == 1)
		a.dmc.SetEnabled(data&0x10 == 1)
        // Writing to $4015 clears the DMC IRQ flag
        a.DmcIRQ = false
        a.dmc.irqPending = false
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
		if p.enabled {
			p.lengthCounter = lengthCounterTable[(data>>3)&0x1F]
		}
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
		if t.enabled {
			t.lengthCounter = lengthCounterTable[(data>>3)&0x1F]
		}
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
		if n.enabled {
			n.lengthCounter = lengthCounterTable[(data>>3)&0x1F]
		}
		n.envelopeStartFlag = true
	}
}

func (d *DMCChannel) cpuWrite(addr uint16, data byte) {
	switch addr {
	case 0x4010:
		d.irqEnabled = (data>>7)&1 == 1
		d.loop = (data>>6)&1 == 1
		d.rateIndex = data & 0x0F
		d.timer = dmcRateTable[d.rateIndex]
	case 0x4011:
		d.outputLevel = data & 0x7F
	case 0x4012:
		d.sampleAddress = 0xC000 + uint16(data)*64
	case 0x4013:
		d.sampleLength = uint16(data)*16 + 1
	}
}
