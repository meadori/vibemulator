package cpu

// Instruction represents a 6502 instruction.
type Instruction struct {
	Name         string
	Operate      func()
	AddrMode     func()
	AddrModeName string
	Cycles       int
}
