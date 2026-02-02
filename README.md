# vibemulator

<p align="center">
  <img src="logo.png" alt="vibemulator logo" width="200"/>
</p>

An NES emulator written in Go, developed with the assistance of the Gemini CLI coding agent.

## Features

*   **CPU:** Emulates the Ricoh 2A03 processor, including all official opcodes.
*   **PPU:** Renders graphics with support for background and sprite rendering.
*   **APU:** Basic audio processing for pulse, triangle, noise, and DMC channels.
*   **Controllers:** Keyboard input for one controller.
*   **Mappers:** Supports NROM (Mapper 0) and MMC1 (Mapper 1) cartridges.
*   **UI:** A simple menu allows for loading ROMs, resetting the console, and exiting the emulator.

## Building

To build the emulator, ensure you have Go (version 1.25.5 or compatible) installed.

```bash
make build
```

The `make build` command will also ensure all Go module dependencies are downloaded.

## Running

To run the emulator, you can optionally provide a `.nes` ROM file as a command-line argument. If no ROM is provided, you can load one via the "LOAD" button in the menu.

```bash
# With a ROM file
make run ROM_FILE=/path/to/your/rom.nes

# Without a ROM file
make run
```

Replace `/path/to/your/rom.nes` with the actual path to your ROM file.

## Testing

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