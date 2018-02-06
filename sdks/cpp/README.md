# C++ Game Server Client SDK

## Usage
Read the [SDK Overview](../), check out [sdk.h](sdk.h) and also look at the
[C++ example](../../examples/cpp-simple).

## Dynamic and Static Libraries
_Note:_ There has yet to be a release of Agones, so if you want static and dynamic libraries,
you will need to [build from source](build/README.md).

If you do not wish to `make build` and build everything, 
the `make` target `build-sdk-cpp` will compile both static and dynamic libraries for Debian/Linux
for your usage, and build two archives that can be untar'd in `/usr/local` (or wherever you like
to keep you .h and .so/.a files) in a `bin` directory inside this one.

- `argonsdk-$(VERSION)-dev-linux-arch_64.tar.gz`: This includes all the 
headers as well as dynamic and static libraries that are needed for development and runtime.
- `argonsdk-$(VERSION)-runtime-linux-arch_64.tar.gz`: This includes just the dynamic libraries that 
are needed to run a binary compiled against the SDK and its dependencies.

## Building From Source
If you wish to compile from source, you will need to compile and install the following dependencies:

- [gRPC](https://grpc.io), v1.8.x - [C++ compilation guide](https://github.com/grpc/grpc/blob/v1.8.x/INSTALL.md)
- [protobuf](https://developers.google.com/protocol-buffers/), v3.5.0 - [C++ compilation guide](https://github.com/google/protobuf/blob/master/src/README.md)

For convenience, it's worth noting that protobuf is include in gRPC's source code, in the `third_party`
directory, and can be compiled from there, rather than being pulling down separately.