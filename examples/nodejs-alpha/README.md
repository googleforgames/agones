# Alpha Node.js Example

This is a very simple alpha "server" that doesn't do much other than show how alpha features of the SDK work in Node.js.

It will:
- Set up the Alpha Agones SDK and connect to the SDK-server
- Add a watch to display status updates from the SDK-server
- Set and get the player capacity (this is not enforced)
- Add, get and remove players, and test if they are present
- Every 10 seconds, write a log showing how long it has been running for
- Every 20 seconds, mark as healthy
- After the shutdown duration (default 60 seconds), shut the server down

To learn how to deploy this example service to GKE, please see the simple server tutorial and adjust as required [Build and Run a Simple Gameserver (node.js)](https://agones.dev/site/docs/tutorials/simple-gameserver-nodejs/).

## Building

If you want to modify the source code and/or build an updated container image, run `make build` from this directory.
This will run the `docker build` command with the correct context.

This example uses a Docker container to host the SDK and example it inside a container so that no special build
tools need to be installed on your system.

## Testing locally with Docker

If you want to run the example locally, you need to start an instance of the SDK-server. To run an SDK-server for
120 seconds, run
```bash
$ cd ../../build; make run-sdk-conformance-local TIMEOUT=120 FEATURE_GATES="PlayerTracking=true" TESTS=ready,watch,health,gameserver
```

In a separate terminal, while the SDK-server is still running, build and start a container with the example gameserver:
```bash
$ make build
$ make run
```

The example can also be run via docker:
```
$ docker run --network=host gcr.io/agones-images/nodejs-alpha-server:0.1
```
Or directly via npm:
```
$ npm start
```

You will see the output like the following:
```
docker run --network=host gcr.io/agones-images/nodejs-alpha-server:0.1

> @ start /home/server/examples/nodejs-alpha
> node src/index.js

Connecting to the SDK server...
...connected to SDK server
```

To see help, pass `--help` as the argument (all are equivalent):
```
$ make args="--help" run
$ docker run --network=host gcr.io/agones-images/nodejs-alpha-server:0.1 --help
$ npm start -- --help
```
