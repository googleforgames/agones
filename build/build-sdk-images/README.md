# Agones SDKs build

As we seen before GameServers can communicate their current state back to Agones controller using a side car. This sidecar runs a GRPC server handling the communication to Agones, the GameServer connects using a SDK which a thin wrapper around GRPC client code.

By using GRPC, adding more languages should be pretty straightforward as it supports code generation for many of them.

> if your language doesn't support GRPC you can still create a SDK by connecting to the HTTP GRPC gateway.

This guide explains how to build, test and generate GRPC code for our SDKs but also how to add a new language.

## How it works

Our build system heavily rely on Docker and make, but that's all you need. The rest of the GRPC tooling and language specific dependencies are installed inside Docker images via Dockerfile.

We separated each language specific tooling in their on Docker image, this way building them will be faster and easier to maintain.

A base GRPC image with protoc is provided, every other images inherit from it via the `FROM` Dockerfile syntax. This way SDKs grpc code is generated from the same version of our SDK sidecar.

## Targets

We currently support 3 commands per SDK:

- `gen` to generate GRPC required by SDKs.
- `test` to run SDKs tests.
- `build` to build SDKs binaries.

> All commands might not be required for all SDKs. (e.g. build is only used for our cpp SDK)

SDKs build scripts and Dockerfile are stored in a folder in this directory.

To run tests for a single SDK use with the `SDK_FOLDER` of your choice:

```bash
make test-sdk SDK_FOLDER=go
```

You can also run all SDKs tests by using `make test-sdks`.

To generate GRPC code and build binaries you can respectively use:

```bash
make gen-sdk-grpc SDK_FOLDER=go
make build-sdk SDK_FOLDER=go

# for all SDKs
make gen-all-sdk-grpc
make build-sdks
```

## Adding support for a new language

Makefile targets run docker containers built from Dockerfile in each folder found in this directory. This means you don't need to change our Makefile

Simply create a new directory with the name of your SDK. Then copy our [template](./tool/template) folder content you SDK folder.

Edit the Dockerfile to install all dependencies you need to generate, build and test your SDK. (you should not need to change the base image)

> As explained in our [SDK documentation](https://agones.dev/site/docs/guides/client-sdks/) you have 2 options HTTP or GRPC. If you are using HTTP you don't need to use our base SDK image so feel free to use the distribution of your choice.

Then add your steps in `build.sh`, `test.sh` and `gen.sh` script files.

You should now be able to use your `SDK_FOLDER` with our [Makefile targets](#targets).

Each targets will ensure that your Dockerfile is built and then run the image with a pre-defined command. The Agones code source repository is mounted in the working directory inside the container and you can also access the current Agones version via the environment variable `VERSION`.