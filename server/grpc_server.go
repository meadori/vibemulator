package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/meadori/vibemulator/api"
	"google.golang.org/grpc"
)

// EmuInterface defines the methods required from the emulator bus for RL
type EmuInterface interface {
	Read(addr uint16) byte
	GetFramePixels() []byte
	LoadState(filename string) error
	Reset()
	SetPaused(bool)
	RequestStep()
	GetCPUState() (a, x, y, sp, p byte, pc uint16, cycles int)
	GetMemoryBlock(addr uint16, size uint16) []byte
}

// GRPCServer manages the network controller connections
type GRPCServer struct {
	api.UnimplementedControllerServiceServer
	mu       sync.Mutex
	P1State  [8]bool
	P2State  [8]bool
	listener net.Listener
	server   *grpc.Server
	emuBus   EmuInterface
}

// NewGRPCServer initializes the gRPC controller server
func NewGRPCServer() *GRPCServer {
	return &GRPCServer{}
}

// SetBus assigns the system bus to the gRPC server for RL memory/frame reads
func (s *GRPCServer) SetBus(b EmuInterface) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emuBus = b
}

// GetFrame returns the raw pixel data from the emulator
func (s *GRPCServer) GetFrame(ctx context.Context, in *api.Empty) (*api.FrameResponse, error) {
	s.mu.Lock()
	bus := s.emuBus
	s.mu.Unlock()

	if bus == nil {
		return nil, fmt.Errorf("emulator bus not connected")
	}
	pixels := bus.GetFramePixels()
	return &api.FrameResponse{Pixels: pixels}, nil
}

// ReadMemory returns the data at a specific memory address in the NES RAM
func (s *GRPCServer) ReadMemory(ctx context.Context, in *api.MemoryRequest) (*api.MemoryResponse, error) {
	s.mu.Lock()
	bus := s.emuBus
	s.mu.Unlock()

	if bus == nil {
		return nil, fmt.Errorf("emulator bus not connected")
	}

	data := bus.Read(uint16(in.Address))
	return &api.MemoryResponse{Data: uint32(data)}, nil
}

// LoadState commands the emulator to load a specific save state file
func (s *GRPCServer) LoadState(ctx context.Context, in *api.StateRequest) (*api.Empty, error) {
	s.mu.Lock()
	bus := s.emuBus
	s.mu.Unlock()

	if bus == nil {
		return nil, fmt.Errorf("emulator bus not connected")
	}

	if err := bus.LoadState(in.Filename); err != nil {
		return nil, fmt.Errorf("failed to load state: %v", err)
	}
	return &api.Empty{}, nil
}

// ResetSystem triggers a hardware reset of the NES, returning to the title screen
func (s *GRPCServer) ResetSystem(ctx context.Context, in *api.Empty) (*api.Empty, error) {
	s.mu.Lock()
	bus := s.emuBus
	s.mu.Unlock()

	if bus == nil {
		return nil, fmt.Errorf("emulator bus not connected")
	}

	bus.Reset()
	return &api.Empty{}, nil
}

// Pause suspends the emulator loop
func (s *GRPCServer) Pause(ctx context.Context, in *api.Empty) (*api.Empty, error) {
	s.mu.Lock()
	bus := s.emuBus
	s.mu.Unlock()

	if bus != nil {
		bus.SetPaused(true)
	}
	return &api.Empty{}, nil
}

// Resume restarts the emulator loop
func (s *GRPCServer) Resume(ctx context.Context, in *api.Empty) (*api.Empty, error) {
	s.mu.Lock()
	bus := s.emuBus
	s.mu.Unlock()

	if bus != nil {
		bus.SetPaused(false)
	}
	return &api.Empty{}, nil
}

// Step advances the CPU by one instruction
func (s *GRPCServer) Step(ctx context.Context, in *api.Empty) (*api.Empty, error) {
	s.mu.Lock()
	bus := s.emuBus
	s.mu.Unlock()

	if bus != nil {
		bus.RequestStep()
	}
	return &api.Empty{}, nil
}

// GetCPUState returns the CPU register values
func (s *GRPCServer) GetCPUState(ctx context.Context, in *api.Empty) (*api.CPUStateResponse, error) {
	s.mu.Lock()
	bus := s.emuBus
	s.mu.Unlock()

	if bus == nil {
		return nil, fmt.Errorf("emulator bus not connected")
	}

	a, x, y, sp, p, pc, cycles := bus.GetCPUState()
	return &api.CPUStateResponse{
		A:      uint32(a),
		X:      uint32(x),
		Y:      uint32(y),
		Sp:     uint32(sp),
		Status: uint32(p),
		Pc:     uint32(pc),
		Cycles: uint32(cycles),
	}, nil
}

// ReadMemoryBlock returns a block of raw NES RAM
func (s *GRPCServer) ReadMemoryBlock(ctx context.Context, in *api.MemoryBlockRequest) (*api.MemoryBlockResponse, error) {
	s.mu.Lock()
	bus := s.emuBus
	s.mu.Unlock()

	if bus == nil {
		return nil, fmt.Errorf("emulator bus not connected")
	}

	block := bus.GetMemoryBlock(uint16(in.Address), uint16(in.Size))
	return &api.MemoryBlockResponse{Data: block}, nil
}


// Start begins listening for gRPC connections on the given port
func (s *GRPCServer) Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = lis
	s.server = grpc.NewServer()
	api.RegisterControllerServiceServer(s.server, s)

	log.Printf("gRPC server listening on :%d", port)

	// Run the server in a background goroutine
	go func() {
		if err := s.server.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the gRPC server
func (s *GRPCServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

// StreamInput handles incoming controller streams from clients
func (s *GRPCServer) StreamInput(stream grpc.BidiStreamingServer[api.InputState, api.Empty]) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		s.mu.Lock()
		state := [8]bool{
			req.A,
			req.B,
			req.Select,
			req.Start,
			req.Up,
			req.Down,
			req.Left,
			req.Right,
		}

		if req.PlayerIndex == 1 || req.PlayerIndex == 0 { // Default to P1 if not specified
			s.P1State = state
		} else if req.PlayerIndex == 2 {
			s.P2State = state
		}
		s.mu.Unlock()
	}
}

// GetP1State returns the current network state for Player 1
func (s *GRPCServer) GetP1State() [8]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.P1State
}

// GetP2State returns the current network state for Player 2
func (s *GRPCServer) GetP2State() [8]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.P2State
}
