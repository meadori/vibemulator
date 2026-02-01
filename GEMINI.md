# Project Overview

`vibemulator` is an NES emulator written in Go. It utilizes the `hajimehoshi/ebiten/v2` library for handling the display and graphics.

# Building and Running

## Requirements
*   Go (version 1.25.5 or compatible)

## Building
To build the emulator executable:
```bash
make build
```
This command will also ensure all Go module dependencies are downloaded.

## Running
To run the emulator, you need to provide a `.nes` ROM file.
```bash
make run ROM_FILE=/path/to/your/rom.nes
```
Alternatively, you can run the compiled executable directly:
```bash
./vibemulator [-debug] /path/to/your/rom.nes
```
The `-debug` flag can be used to enable debug logging.

# Testing

## Running all tests
To execute all project tests:
```bash
make test
```

## Running NESTest CPU tests
For specific CPU tests based on the NESTest ROM:
```bash
make nestest
```

# Development Conventions

## Go Version
The project is developed with Go version `1.25.5`. It is recommended to use this version to avoid potential build or compatibility issues.

## Task Automation
A `Makefile` is used to automate common development tasks such as building, running, testing, and cleaning.

## Dependency Management
Go modules (`go.mod`, `go.sum`) are used for dependency management. Dependencies are automatically downloaded by `make build` or `make test`.

# Directory Structure

The project is organized into several packages, each responsible for a specific component of the NES emulation:
*   `apu/`: Audio Processing Unit
*   `bus/`: Main system bus for communication between components
*   `cartridge/`: Handles ROM loading and cartridge specific logic (e.g., NROM mapper)
*   `controller/`: Input handling
*   `cpu/`: Central Processing Unit (Ricoh 2A03) emulation
*   `display/`: Graphics and display handling using Ebiten
*   `docs/`: Project documentation
*   `mapper/`: ROM mapper interfaces
*   `nestest/`: Tools and logic for running the NESTest ROM for CPU verification
*   `ppu/`: Picture Processing Unit (graphics and rendering)
