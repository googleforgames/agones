# Simple Rust Example

This is a very simple "server" that doesn't do much other than show how the SDK works in Rust.

It will
- Setup the Agones SDK
- Call `SDK::Ready()` to register that it is ready with Agones.
- Every 10 seconds, write a log saying "Hi! I'm a Game Server"
- After 60 seconds, call `SDK::Shutdown()` to shut the server down.

To learn how to deploy this example service to GKE, please see the tutorial [Build and Run a Simple Gameserver (Rust)](https://agones.dev/site/docs/tutorials/simple-gameserver-rust/).

## Building

If you have a local rust developer environment installed locally, you can run `make build-server` to compile the code and
`make run` to execute the resulting binary.

If you want to build an updated container image or want to build the source code without installing the rust developer
tools locally, run `make build-image` to run the `docker build` command with the correct context.

This example uses the [Docker builder pattern](https://docs.docker.com/develop/develop-images/multistage-build/) to
build the SDK, example and host it inside a container.

## Testing locally with Docker

If you want to run the example locally, you need to start an instance of the SDK-server. To run an SDK-server for
120 seconds, run
```bash
$ cd ../../build; make run-sdk-conformance-local TIMEOUT=120 TESTS=ready,watch,health,gameserver
```

In a separate terminal, while the SDK-server is still running, build and start a container with the example gameserver:
```bash
$ make build-image
$ make run-image
```

You will see the following output:
```
docker run --network=host us-docker.pkg.dev/agones-images/examples/rust-simple-server:0.11
Rust Game Server has started!
Creating SDK instance
Setting a label
Starting to watch GameServer updates...
Health ping sent
Setting an annotation
GameServer Update, name: local
GameServer Update, state: Ready
Marking server as ready...
...marked Ready
Getting GameServer details...
GameServer name: local
Running for 0 seconds
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 10 seconds
```

When you are finished, clean up the `sdk` directory:
```
make clean-docker
```

## Testing locally (without Docker)

If you want to run the example locally, you need to start an instance of the SDK-server. To run an SDK-server for
120 seconds, run
```bash
$ cd ../../build; make run-sdk-conformance-local TIMEOUT=120 TESTS=ready,watch,health,gameserver
```

In a separate terminal, while the SDK-server is still running, build and execute the example gameserver:
```bash
$ make build
$ make run
```

You will see the following output:
```
Rust Game Server has started!
Creating SDK instance
Setting a label
Starting to watch GameServer updates...
Health ping sent
Setting an annotation
...
```

When you are finished, clean up the `sdk` directory and `target` folder:
```
make clean
```
