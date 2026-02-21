package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/meadori/vibemulator/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func parseButtons(fullStr string) (int, *api.InputState) {
	parts := strings.Split(fullStr, ":")
	if len(parts) != 2 {
		return 0, &api.InputState{}
	}
	
	playerIndex := 1
	if parts[0] == "P2" {
		playerIndex = 2
	}
	
	buttonStr := parts[1]
	state := &api.InputState{PlayerIndex: int32(playerIndex)}
	if buttonStr == "NONE" {
		return playerIndex, state
	}
	
	buttons := strings.Split(buttonStr, "+")
	for _, b := range buttons {
		switch strings.ToUpper(b) {
		case "A":      state.A = true
		case "B":      state.B = true
		case "SELECT": state.Select = true
		case "START":  state.Start = true
		case "UP":     state.Up = true
		case "DOWN":   state.Down = true
		case "LEFT":   state.Left = true
		case "RIGHT":  state.Right = true
		}
	}
	return playerIndex, state
}

func main() {
	scriptFile := flag.String("script", "", "Path to the recorded script file to replay")
	flag.Parse()

	if *scriptFile == "" {
		log.Fatalf("Please provide a script file using -script <file.script>")
	}

	file, err := os.Open(*scriptFile)
	if err != nil {
		log.Fatalf("Failed to open script file: %v", err)
	}
	defer file.Close()

	// 1. Connect to the emulator's gRPC server
	log.Println("Connecting to emulator on localhost:50051...")
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := api.NewControllerServiceClient(conn)

	// 2. Open a streaming connection
	stream, err := client.StreamInput(context.Background())
	if err != nil {
		log.Fatalf("failed to open stream: %v", err)
	}

	log.Printf("Connected! Starting replay of %s in 2 seconds...\n", *scriptFile)
	time.Sleep(2 * time.Second)

	// 3. Read and execute the script
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			log.Printf("Skipping invalid line: %s\n", line)
			continue
		}

		frames, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Printf("Invalid frame count: %s\n", parts[0])
			continue
		}

		// Replay all player states on this line
		for i := 1; i < len(parts); i++ {
			_, state := parseButtons(parts[i])
			// Send state for each player
			if err := stream.Send(state); err != nil {
				log.Fatalf("failed to send state: %v", err)
			}
		}
		
		// 1 frame = ~16.666ms at 60Hz
		duration := time.Duration(float64(frames) * 16.666666 * float64(time.Millisecond))
		
		// Wait for the duration
		time.Sleep(duration)
	}

	// Gracefully close the send stream
	if err := stream.CloseSend(); err != nil {
		log.Printf("failed to close stream: %v", err)
	}

	log.Println("Replay complete. Disconnected.")
}
