package cpu

type State struct {
	PC, AddrAbs, AddrRel            uint16
	SP, A, X, Y, P, Opcode, Fetched byte
	Cycles                          int
	NmiPending, IrqPending          bool
}

func (c *CPU) SaveState() State {
	return State{c.PC, c.addrAbs, c.addrRel, c.SP, c.A, c.X, c.Y, c.P, c.opcode, c.fetched, c.Cycles, c.nmiPending, c.irqPending}
}

func (c *CPU) LoadState(s State) {
	c.PC, c.addrAbs, c.addrRel, c.SP, c.A, c.X, c.Y, c.P, c.opcode, c.fetched, c.Cycles, c.nmiPending, c.irqPending = s.PC, s.AddrAbs, s.AddrRel, s.SP, s.A, s.X, s.Y, s.P, s.Opcode, s.Fetched, s.Cycles, s.NmiPending, s.IrqPending
}
