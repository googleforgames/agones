# Build

Tooling for building and developing against Agon, with only dependency being [Docker](https://www.docker.com).

Rather than installing all the dependencies locally, you can test and build Agon using the Docker image that can
be build from the Dockerfile based image in this directory. There is an accompanying Makefile for all the common
tasks you may wish to accomplish.

## Make Targets

### `make shell`
Run a bash shell with the developer tools ad source code in it.
Also creates the image if it doesn't exist

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
