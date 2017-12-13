# Build

Tooling for building and developing against Agon, with only dependencies being
[Make](https://www.gnu.org/software/make/) and [Docker](https://www.docker.com)

Rather than installing all the dependencies locally, you can test and build Agon using the Docker image that is
built from the Dockerfile in this directory. There is an accompanying Makefile for all the common
tasks you may wish to accomplish.

## GOPATH

This project should be cloned to the directory `$GOPATH/src/github.com/agonio/agon`
for when you are developing locally, and require package resolution in your IDE.

This is not required if you are simply building using the `make` targets

## Make Targets

All targets will create the build image if it is not present.

### Development Targets

Targets for developing with the build image

#### `make build`
Build all the images required for Agon

#### `make test`
Run all tests

### `make shell`
Run a bash shell with the developer tools (go tooling, kubectl, etc) and source code in it.

### `make godoc`
Run a container with godoc (search index enabled)

#### `make build-gameservers-controller-image`
Compile the gameserver controller and then build the docker image

#### `make build-gameservers-sidecar-image`
Compile the gameserver sidecar and then build the docker image

#### `make gen-crd-client`
Generate the Custom Resource Definition client(s)

#### `make gen-gameservers-sidecar-grpc`
Generate the gRPC sidecar Server and Client

### Build Image Targets

Targets for building the build image

### `make clean-config`
Cleans the kubernetes and gcloud configurations

### `make clean-image`
Deletes the local build docker image

### `make build-image`
Creates the build docker image

## Google Cloud Platform

A set of utilities for setting up a Container Engine cluster on Google Cloud Platform,
since it's an easy way to get a test cluster working with Kubernetes.

### `make gcloud-init`
Initialise the gcloud login and project configuration, if you are working with GCP

### `make gcloud-test-cluster`
Creates and authenticates a small, 3 node GKE cluster to work against

### `make gcloud-auth-cluster`
Pulls down authentication information for kubectl against a cluster, name can be specified through CLUSTER_NAME
(defaults to 'test-cluster')
