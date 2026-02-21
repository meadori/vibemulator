package mapper

// Mapper defines the interface for different NES mappers.
type Mapper interface {
	CPUMapRead(addr uint16) (byte, bool)
	CPUMapWrite(addr uint16, data byte) bool
	PPUMapRead(addr uint16) (byte, bool)
	PPUMapWrite(addr uint16, data byte) bool
	GetMirroring() byte
	Clock()
	IRQPending() bool
	ClearIRQ()
	Save() []byte
	Load([]byte) error
}
