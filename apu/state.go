package apu

type PulseState struct {
	Enabled, IsPulse1, LengthCounterHalt, ConstantVolume, SweepEnabled, SweepNegate, SweepReloadFlag, EnvelopeStartFlag                      bool
	DutyCycle, Volume, SweepPeriod, SweepShift, LengthCounter, DutySequencer, SweepCounter, EnvelopeVolume, EnvelopeDivider, EnvelopeCounter byte
	Timer, TimerCounter                                                                                                                      uint16
}

type TriangleState struct {
	Enabled, LengthCounterHalt, LinearCounterReloadFlag            bool
	LinearCounterLoad, LinearCounter, LengthCounter, DutySequencer byte
	Timer, TimerCounter                                            uint16
}

type NoiseState struct {
	Enabled, LengthCounterHalt, ConstantVolume, Mode, EnvelopeStartFlag                  bool
	Volume, TimerPeriod, LengthCounter, EnvelopeVolume, EnvelopeDivider, EnvelopeCounter byte
	ShiftRegister, TimerCounter                                                          uint16
}

type DMCState struct {
	Enabled, IrqEnabled, Loop, SampleBufferEmpty, SilenceFlag, IrqPending bool
	RateIndex, OutputLevel, ShiftRegister, BitsRemaining, SampleBuffer    byte
	Timer, SampleAddress, SampleLength, CurrentAddress, BytesRemaining    uint16
}

type State struct {
	Pulse1                          PulseState
	Pulse2                          PulseState
	Triangle                        TriangleState
	Noise                           NoiseState
	DMC                             DMCState
	Cycle, FrameCounter             uint64
	FrameSequenceStep, SequenceMode byte
	IrqInhibit, DmcIRQ, FrameIRQ    bool
	SampleCycleCounter              float64
}

func (p *PulseChannel) SaveState() PulseState {
	return PulseState{p.enabled, p.isPulse1, p.lengthCounterHalt, p.constantVolume, p.sweepEnabled, p.sweepNegate, p.sweepReloadFlag, p.envelopeStartFlag, p.dutyCycle, p.volume, p.sweepPeriod, p.sweepShift, p.lengthCounter, p.dutySequencer, p.sweepCounter, p.envelopeVolume, p.envelopeDivider, p.envelopeCounter, p.timer, p.timerCounter}
}

func (p *PulseChannel) LoadState(s PulseState) {
	p.enabled, p.isPulse1, p.lengthCounterHalt, p.constantVolume, p.sweepEnabled, p.sweepNegate, p.sweepReloadFlag, p.envelopeStartFlag = s.Enabled, s.IsPulse1, s.LengthCounterHalt, s.ConstantVolume, s.SweepEnabled, s.SweepNegate, s.SweepReloadFlag, s.EnvelopeStartFlag
	p.dutyCycle, p.volume, p.sweepPeriod, p.sweepShift, p.lengthCounter, p.dutySequencer, p.sweepCounter, p.envelopeVolume, p.envelopeDivider, p.envelopeCounter = s.DutyCycle, s.Volume, s.SweepPeriod, s.SweepShift, s.LengthCounter, s.DutySequencer, s.SweepCounter, s.EnvelopeVolume, s.EnvelopeDivider, s.EnvelopeCounter
	p.timer, p.timerCounter = s.Timer, s.TimerCounter
}

func (t *TriangleChannel) SaveState() TriangleState {
	return TriangleState{t.enabled, t.lengthCounterHalt, t.linearCounterReloadFlag, t.linearCounterLoad, t.linearCounter, t.lengthCounter, t.dutySequencer, t.timer, t.timerCounter}
}

func (t *TriangleChannel) LoadState(s TriangleState) {
	t.enabled, t.lengthCounterHalt, t.linearCounterReloadFlag, t.linearCounterLoad, t.linearCounter, t.lengthCounter, t.dutySequencer, t.timer, t.timerCounter = s.Enabled, s.LengthCounterHalt, s.LinearCounterReloadFlag, s.LinearCounterLoad, s.LinearCounter, s.LengthCounter, s.DutySequencer, s.Timer, s.TimerCounter
}

func (n *NoiseChannel) SaveState() NoiseState {
	return NoiseState{n.enabled, n.lengthCounterHalt, n.constantVolume, n.mode, n.envelopeStartFlag, n.volume, n.timerPeriod, n.lengthCounter, n.envelopeVolume, n.envelopeDivider, n.envelopeCounter, n.shiftRegister, n.timerCounter}
}

func (n *NoiseChannel) LoadState(s NoiseState) {
	n.enabled, n.lengthCounterHalt, n.constantVolume, n.mode, n.envelopeStartFlag, n.volume, n.timerPeriod, n.lengthCounter, n.envelopeVolume, n.envelopeDivider, n.envelopeCounter, n.shiftRegister, n.timerCounter = s.Enabled, s.LengthCounterHalt, s.ConstantVolume, s.Mode, s.EnvelopeStartFlag, s.Volume, s.TimerPeriod, s.LengthCounter, s.EnvelopeVolume, s.EnvelopeDivider, s.EnvelopeCounter, s.ShiftRegister, s.TimerCounter
}

func (d *DMCChannel) SaveState() DMCState {
	return DMCState{d.enabled, d.irqEnabled, d.loop, d.sampleBufferEmpty, d.silenceFlag, d.irqPending, d.rateIndex, d.outputLevel, d.shiftRegister, d.bitsRemaining, d.sampleBuffer, d.timer, d.sampleAddress, d.sampleLength, d.currentAddress, d.bytesRemaining}
}

func (d *DMCChannel) LoadState(s DMCState) {
	d.enabled, d.irqEnabled, d.loop, d.sampleBufferEmpty, d.silenceFlag, d.irqPending, d.rateIndex, d.outputLevel, d.shiftRegister, d.bitsRemaining, d.sampleBuffer, d.timer, d.sampleAddress, d.sampleLength, d.currentAddress, d.bytesRemaining = s.Enabled, s.IrqEnabled, s.Loop, s.SampleBufferEmpty, s.SilenceFlag, s.IrqPending, s.RateIndex, s.OutputLevel, s.ShiftRegister, s.BitsRemaining, s.SampleBuffer, s.Timer, s.SampleAddress, s.SampleLength, s.CurrentAddress, s.BytesRemaining
}

func (a *APU) SaveState() State {
	return State{a.pulse1.SaveState(), a.pulse2.SaveState(), a.triangle.SaveState(), a.noise.SaveState(), a.dmc.SaveState(), a.cycle, a.frameCounter, a.frameSequenceStep, a.sequenceMode, a.irqInhibit, a.DmcIRQ, a.FrameIRQ, a.sampleCycleCounter}
}

func (a *APU) LoadState(s State) {
	a.pulse1.LoadState(s.Pulse1)
	a.pulse2.LoadState(s.Pulse2)
	a.triangle.LoadState(s.Triangle)
	a.noise.LoadState(s.Noise)
	a.dmc.LoadState(s.DMC)
	a.cycle, a.frameCounter, a.frameSequenceStep, a.sequenceMode, a.irqInhibit, a.DmcIRQ, a.FrameIRQ, a.sampleCycleCounter = s.Cycle, s.FrameCounter, s.FrameSequenceStep, s.SequenceMode, s.IrqInhibit, s.DmcIRQ, s.FrameIRQ, s.SampleCycleCounter
}
