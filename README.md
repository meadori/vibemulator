# vibemulator

<p align="center">
  <img src="logo.png" alt="vibemulator logo" width="400"/>
</p>

An NES emulator written in Go, developed with the assistance of the Gemini CLI coding agent.

## Features

*   **CPU:** Emulates the Ricoh 2A03 processor, including all official opcodes.
*   **PPU:** Renders graphics with support for background and sprite rendering.
*   **APU:** Basic audio processing for pulse, triangle, noise, and DMC channels.
*   **Controllers:** Supports both local keyboard input and remote gRPC-based network controllers.
*   **Mappers:** Supports NROM (Mapper 0) and MMC1 (Mapper 1) cartridges.
*   **UI:** A custom-styled menu bar with 3D beveled buttons, a glowing power LED, and NES-inspired branding.
*   **Scripting:** Built-in macro recording and replaying capabilities.

## Building

To build the emulator and the gRPC toolchain, ensure you have Go (version 1.25.5 or compatible) installed.

```bash
make build
```

## Running

To run the emulator, you can optionally provide a `.nes` ROM file as a command-line argument or load one via the **LOAD** button in the top menu.

```bash
# Standard run
make run ROM_FILE=/path/to/rom.nes

# With debug logging enabled
./vibemulator -debug /path/to/rom.nes
```

### Controls (Player 1)
- **Arrows:** Directional Pad
- **Z:** A Button
- **X:** B Button
- **Enter:** Start
- **Shift:** Select

### Save States
- **F5:** Save State to `vibemulator.sav`
- **F7:** Load State from `vibemulator.sav`

### Time Rewind
- **Backspace:** Hold down to instantly reverse gameplay in real-time. Release to resume playing from the newly rewound point! (Buffer holds roughly 20 seconds of history).

### Debugger
- **Tab:** Toggle PPU Pattern Table Viewer
- **P:** Cycle active palette (0-7) when the Viewer is open

## Network Play & Scripting

Vibemulator includes a built-in gRPC server (port 50051) that allows remote clients to stream controller inputs to the emulator over a network.

### Macro Recording
You can record your gameplay to a human-readable script file for later analysis or replay.

```bash
# Record gameplay to "mysession.script"
./vibemulator -record mysession.script /path/to/rom.nes
```

### Macro Replay (via gRPC)
You can replay a recorded session by streaming the script through the provided gRPC client.

1.  Start the emulator normally: `./vibemulator /path/to/rom.nes`
2.  In a separate terminal, run the replayer:
    ```bash
    go run cmd/client/main.go -script mysession.script
    ```

### Reinforcement Learning (DQN)

Vibemulator features a built-in interface for training Reinforcement Learning (RL) agents, specifically modeled after the famous Deep Q-Network (DQN) architecture that learned to play Atari games from raw pixels.

A `Gymnasium` compatible Python environment connects to the Go emulator via gRPC. This allows the Python agent to receive raw frame buffers (screen pixels), send controller button presses, and read raw NES RAM to calculate custom rewards (e.g., Mario's score or position).

### Setup the RL Environment

To set up the Python virtual environment, install the data-science dependencies (`torch`, `gymnasium`, `numpy`, `grpcio`), and compile the protocol buffers, run:

```bash
make rl-setup
```

### Training an Agent

1. First, start the emulator with a ROM (e.g., Super Mario Bros):
   ```bash
   ./vibemulator /path/to/super_mario_bros.nes
   ```

2. In a separate terminal, launch the PyTorch training loop:
   ```bash
   make rl-train
   ```

*(Note: You will need to customize the memory addresses read in `rl/vibemulator_env.py` to match the specific RAM map of the game you are trying to train.)*

# Testing

To run the tests, use the following command:

```bash
make test
```

## Cleaning

To remove build artifacts and clear Go cache:

```bash
make clean
```

## Development

This project was developed entirely using the Gemini CLI coding agent, an experimental tool from Google. The agent was responsible for writing, debugging, and committing the code based on high-level user prompts.

### Development Conventions

- **Code Formatting:** Always run `make fmt` before committing to ensure consistent code style.