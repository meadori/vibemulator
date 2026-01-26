# vibemulator

An NES emulator written in Go.

## Building

To build the emulator, you will need to have [Bazel](https://bazel.build/) installed.

```bash
bazel build //:vibemulator
```

## Running

```bash
bazel run //:vibemulator
```

## Testing

To run the tests, use the following command:

```bash
bazel test //...
```
