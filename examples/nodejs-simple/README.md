# Simple node.js Example

This is a very simple "server" that doesn't do much other than show how the SDK works in node.js.

It will
- Setup the Agones SDK
- Call `sdk.ready()` to register that it is ready with Agones.
- Every 10 seconds, write a log showing how long it has been running for
- After 60 seconds, call `sdk.shutdown()` to shut the server down.

To learn how to deploy this example service to GKE, please see the tutorial [Build and Run a Simple Gameserver (node.js)](https://agones.dev/site/docs/tutorials/simple-gameserver-nodejs/).

## Building

If you want to modify the source code and/or build an updated container image, run `make build` from this directory.
This will run the `docker build` command with the correct context.

This example uses a Docker container to host the SDK and example it inside a container so that no special build
tools need to be installed on your system.

## Testing locally with Docker

If you want to run the example locally, you need to start an instance of the SDK-server. To run an SDK-server for
120 seconds, run
```bash
$ cd ../../build; make run-sdk-conformance-local TIMEOUT=120 TESTS=ready,watch,health,gameserver
```

In a separate terminal, while the SDK-server is still running, build and start a container with the example gameserver:
```bash
$ make build
$ make run
```

You will see the output like the following:
```
docker run --network=host gcr.io/agones-images/nodejs-simple-server:0.1

> @ start /home/server/examples/nodejs-simple
> node src/index.js

node.js Game Server has started!
Setting a label
(node:18) [DEP0005] DeprecationWarning: Buffer() is deprecated due to security and usability issues. Please use the Buffer.alloc(), Buffer.allocUnsafe(), or Buffer.from() methods instead.
Setting an annotation
GameServer Update:
	name: local 
	state: Ready
GameServer Update:
	name: local 
	state: Ready
Marking server as ready...
...marked Ready
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 10 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 20 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 30 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 40 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 50 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Shutting down after 60 seconds...
...marked for Shutdown
Running for 60 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 70 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 80 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 90 seconds!
```
