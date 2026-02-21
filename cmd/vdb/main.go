package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/meadori/vibemulator/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	fmt.Println("VDB - Vibemulator DeBugger")
	fmt.Println("Connecting to emulator on localhost:50051...")

	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := api.NewControllerServiceClient(conn)
	fmt.Println("Connected. Type 'help' for commands.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("(vdb) ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]

		switch cmd {
		case "help", "h":
			fmt.Println("Commands:")
			fmt.Println("  run, c      - Resume execution")
			fmt.Println("  pause, p    - Pause execution")
			fmt.Println("  step, s     - Step one instruction")
			fmt.Println("  regs, i r   - Print CPU registers")
			fmt.Println("  x <addr>    - Examine memory (e.g. x 0000 or x/16 0000)")
			fmt.Println("  quit, q     - Exit debugger")
		case "quit", "q", "exit":
			return
		case "pause", "p":
			_, err := client.Pause(context.Background(), &api.Empty{})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Emulator paused.")
				printRegs(client)
			}
		case "run", "c", "continue":
			_, err := client.Resume(context.Background(), &api.Empty{})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("Emulator running...")
			}
		case "step", "s":
			_, err := client.Step(context.Background(), &api.Empty{})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				printRegs(client)
			}
		case "regs", "i":
			if len(parts) > 1 && parts[1] == "r" || cmd == "regs" {
				printRegs(client)
			} else {
				fmt.Println("Unknown command. Did you mean 'i r'?")
			}
		case "x":
			count := 1
			addrStr := ""
			if len(parts) == 1 {
				fmt.Println("Usage: x <addr> or x/<count> <addr>")
				continue
			} else if strings.HasPrefix(parts[0], "x/") {
				countStr := strings.TrimPrefix(parts[0], "x/")
				parsedCount, err := strconv.ParseInt(countStr, 10, 32)
				if err == nil {
					count = int(parsedCount)
				}
				addrStr = parts[1]
			} else {
				addrStr = parts[1]
			}

			// Clean up address (e.g., remove 0x prefix if present)
			addrStr = strings.TrimPrefix(addrStr, "0x")
			addr, err := strconv.ParseUint(addrStr, 16, 32)
			if err != nil {
				fmt.Printf("Invalid address: %s\n", parts[1])
				continue
			}

			res, err := client.ReadMemoryBlock(context.Background(), &api.MemoryBlockRequest{
				Address: uint32(addr),
				Size:    uint32(count),
			})
			if err != nil {
				fmt.Printf("Error reading memory: %v\n", err)
			} else {
				printHexDump(uint16(addr), res.Data)
			}
		default:
			// check for x/count without space like x/10 0x0000
			if strings.HasPrefix(cmd, "x/") {
				countStr := strings.TrimPrefix(cmd, "x/")
				count, _ := strconv.ParseInt(countStr, 10, 32)
				if count <= 0 {
					count = 1
				}
				if len(parts) > 1 {
					addrStr := strings.TrimPrefix(parts[1], "0x")
					addr, err := strconv.ParseUint(addrStr, 16, 32)
					if err != nil {
						fmt.Printf("Invalid address: %s\n", parts[1])
						continue
					}
					res, err := client.ReadMemoryBlock(context.Background(), &api.MemoryBlockRequest{
						Address: uint32(addr),
						Size:    uint32(count),
					})
					if err != nil {
						fmt.Printf("Error: %v\n", err)
					} else {
						printHexDump(uint16(addr), res.Data)
					}
				}
			} else {
				fmt.Printf("Unknown command: %s\n", cmd)
			}
		}
	}
}

func printRegs(client api.ControllerServiceClient) {
	state, err := client.GetCPUState(context.Background(), &api.Empty{})
	if err != nil {
		fmt.Printf("Error getting CPU state: %v\n", err)
		return
	}
	fmt.Printf("A: %02X  X: %02X  Y: %02X  SP: %02X  PC: %04X  Status: %02b\n",
		state.A, state.X, state.Y, state.Sp, state.Pc, state.Status)
}

func printHexDump(startAddr uint16, data []byte) {
	for i := 0; i < len(data); i += 16 {
		fmt.Printf("%04X:", startAddr+uint16(i))
		end := i + 16
		if end > len(data) {
			end = len(data)
		}
		for j := i; j < end; j++ {
			fmt.Printf(" %02X", data[j])
		}
		fmt.Println()
	}
}
