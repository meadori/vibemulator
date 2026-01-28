package cpu

// Instruction represents a 6502 instruction.
type Instruction struct {
	Name         string
	Operate      func() byte
	AddrMode     func() byte
	AddrModeName string
	Cycles       int
}
