# Project Plan: vibemulator

This document outlines the development plan for the vibemulator NES emulator.

## Milestones

### Milestone 1: Basic CPU Implementation

- [x] Implement CPU memory map (RAM)
- [x] Implement CPU instruction set (subset for basic ROMs)
- [x] Implement all official CPU instructions
- [x] Load and execute a simple ROM

### Milestone 2: Cartridge and Mapper Support

- [x] Load .nes file format
- [ ] Implement Mapper 0 (NROM)

### Milestone 3: PPU and Graphics

- [ ] Implement PPU registers
- [ ] Implement PPU memory map (VRAM, palettes)
- [ ] Render a basic frame
- [ ] Implement sprites and background rendering

### Milestone 3: Input and Audio

- [ ] Implement controller support
- [ ] Implement APU and audio output

### Milestone 4: Advanced Features and Compatibility

- [ ] Implement full CPU instruction set
- [ ] Implement mappers for advanced ROMs
- [ ] Improve performance and accuracy
- [ ] Save/load state
- [ ] Debugging tools

## Feature Checklist

### CPU

- [x] Registers (PC, SP, A, X, Y, P)
- [x] Addressing Modes
- [x] Instructions (subset)
- [x] Instructions (full)
- [ ] Interrupts (NMI, IRQ, BRK)

### PPU

- [ ] Registers
- [ ] VRAM
- [ ] Palettes
- [ ] Background rendering
- [ ] Sprite rendering
- [ ] Scrolling

### APU

- [ ] Pulse channels
- [ ] Triangle channel
- [ ] Noise channel
- [ ] DMC channel

### Input

- [ ] Standard controllers

### Mappers

- [ ] NROM (Mapper 0)
- [ ] MMC1 (Mapper 1)
- [ ] UXROM (Mapper 2)
- [ ] CNROM (Mapper 3)
- [ ] MMC3 (Mapper 4)
