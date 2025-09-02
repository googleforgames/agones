# Autoscaler WASM example

This example demonstrates how to build and run a WebAssembly (WASM) plugin that implements a simple Agones FleetAutoscaler policy using the Extism PDK for Go.

The plugin exports two WASM functions:
- scale: Ensures there is a buffer of replicas available for the fleet. The buffer size is configurable via the buffer_size config entry and defaults to 5.
- scaleNone: A no-op example that always returns the current replica count.

## Prerequisites
- Go 1.24+ with WASI support (GOOS=wasip1, GOARCH=wasm)
- extism CLI installed and available in PATH (for local testing)
- Make

## Files
- main.go: The WASM plugin implementation using github.com/extism/go-pdk.
- model.go: Minimal request/response models mirroring the FleetAutoscaler webhook types used by Agones.
- Makefile: Convenience targets to build and locally test the WASM module.

## Build
To compile the WASM plugin:

make build

This produces plugin.wasm in the current directory.

## Run the scale function with a sample request
make test

The Makefile passes a minimal FleetAutoscaleReview JSON via --input and runs the exported scale function. You can override arguments by supplying ARGS. For example, to call the alternate export or set config values:

## `./build/make shell`
If you run `make shell` from the ./build directory, the required extism CLI will be available in the PATH for you
along with the required Go version.
