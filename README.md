# vibemulator

An NES emulator written in Go.

## Building

To build the emulator, you will need to have [Bazel](https://bazel.build/) installed.

```bash
bazel build //:vibemulator
```

## Running

To run the emulator, you will need a `.nes` ROM file.

```bash
bazel run //:vibemulator -- /path/to/your/rom.nes
```

Replace `/path/to/your/rom.nes` with the actual path to your ROM file.

## Testing

To run the tests, use the following command:

```bash
bazel test //...
```
