# Simple node.js Example

This is a very simple "server" that doesn't do much other than show how the SDK works in node.js.

It will:
- Set up the Agones SDK and connect to the SDK-server
- Add a watch to display status updates from the SDK-server
- Set labels, annotations
- Manage a fake lifecycle from `Ready` through `Allocated` and `Reserved`
- Every 10 seconds, write a log showing how long it has been running for
- Every 20 seconds, mark as healthy
- After the shutdown duration (default 60 seconds), shut the server down

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

Started with shutdown duration of 60 seconds. Connecting to the SDK server...
...connected to SDK server
Running for 10 seconds
Setting a label
GameServer Update:
        name: local
        state: Ready
        labels: agones.dev/sdk-test-label,test-value & islocal,true
        annotations: annotation,true
Health ping sent
Running for 20 seconds
Setting an annotation
GameServer Update:   
        name: local
        state: Ready
        labels: agones.dev/sdk-test-label,test-value & islocal,true
        annotations: agones.dev/sdk-test-annotation,test value & annotation,true
Running for 30 seconds
Marking server as ready...
GameServer Update:
        name: local
        state: Ready
        labels: agones.dev/sdk-test-label,test-value & islocal,true
        annotations: agones.dev/sdk-test-annotation,test value & annotation,true
Health ping sent
Running for 40 seconds
Allocating
GameServer Update:
        name: local
        state: Allocated
        labels: agones.dev/sdk-test-label,test-value & islocal,true
        annotations: agones.dev/sdk-test-annotation,test value & annotation,true
Running for 50 seconds
Reserving for 10 seconds
GameServer Update:
        name: local
        state: Reserved
        labels: agones.dev/sdk-test-label,test-value & islocal,true
        annotations: agones.dev/sdk-test-annotation,test value & annotation,true
Health ping sent
Running for 60 seconds
GameServer Update:
        name: local
        state: Ready
        labels: agones.dev/sdk-test-label,test-value & islocal,true
        annotations: agones.dev/sdk-test-annotation,test value & annotation,true
Running for 70 seconds
Shutting down after 60 seconds...
Health ping sent
Running for 80 seconds
Running for 90 seconds
Health ping sent
Running for 100 seconds
Running for 110 seconds
Health ping sent
Running for 120 seconds
Running for 130 seconds
Shutting down...
GameServer Update:
        name: local
        state: Shutdown
        labels: agones.dev/sdk-test-label,test-value & islocal,true
        annotations: agones.dev/sdk-test-annotation,test value & annotation,true
Health ping sent
Running for 140 seconds
Closing connection to SDK server
Running for 150 seconds
Exiting

```

You can optionally specify how long the server will stay up once the basic tests are complete.
To do this pass arguments through, e.g. to increase the shutdown duration to 120 seconds:
```
$ make args="120" run
```

Please note that there is a max sleep time in Node.js of 2,147,483 seconds (~24 days). Values will be capped at this and negative values or zero will also be set to this.
So to keep the server alive for a long time you could use the following:
```
$ make args="0" run
```

To see help pass `--help` as the argument:
```
$ make args="--help" run
```

If you are not using make and running directly then the equivalent commands are:
```
node start
node start -- 120
node start -- 0
node start -- --help
```