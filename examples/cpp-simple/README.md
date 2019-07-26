# Simple C++ Example

This is a very simple "server" that doesn't do much other than show how the SDK works in C++.

It will
- Setup the Agones SDK
- Call `SDK::Ready()` to register that it is ready with Agones.
- Every 10 seconds, write a log showing how long it has been running for
- After 60 seconds, call `SDK::Shutdown()` to shut the server down.

To learn how to deploy this example service to GKE, please see the tutorial [Build and Run a Simple Gameserver (C++)](https://agones.dev/site/docs/tutorials/simple-gameserver-cpp/).

## Building

If you want to modify the source code and/or build an updated container image, run `make build` from this directory.
This will run the `docker build` command with the correct context.

This example uses the [Docker builder pattern](https://docs.docker.com/develop/develop-images/multistage-build/) to
build the SDK, example and host it inside a container.