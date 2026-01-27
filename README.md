# vibemulator

An NES emulator written in Go.

## Building

To build the emulator, ensure you have Go (version 1.25.5 or compatible) installed.

```bash
make build
```

The `make build` command will also ensure all Go module dependencies are downloaded.

## Running

To run the emulator, you will need a `.nes` ROM file.

```bash
make run ROM_FILE=/path/to/your/rom.nes
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
