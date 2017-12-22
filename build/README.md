# Developing, Testing and Building Agon

Tooling for building and developing against Agon, with only dependencies being
[Make](https://www.gnu.org/software/make/) and [Docker](https://www.docker.com)

Rather than installing all the dependencies locally, you can test and build Agon using the Docker image that is
built from the Dockerfile in this directory. There is an accompanying Makefile for all the common
tasks you may wish to accomplish.

<!-- ToC start -->
## Table of Contents

   1. [GOPATH](#gopath)
   1. [Testing and Building](#testing-and-building)
      1. [Running a Test Google Kubernetes Engine Cluster](#running-a-test-google-kubernetes-engine-cluster)
      1. [Running a Test Minikube cluster](#running-a-test-minikube-cluster)
      1. [Next Steps](#next-steps)
   1. [Make Variable Reference](#make-variable-reference)
      1. [VERSION](#version)
      1. [REGISTRY](#registry)
      1. [KUBECONFIG](#kubeconfig)
      1. [CLUSTER_NAME](#cluster_name)
   1. [Make Target Reference](#make-target-reference)
      1. [Development Targets](#development-targets)
      1. [Build Image Targets](#build-image-targets)
      1. [Google Cloud Platform](#google-cloud-platform)
<!-- ToC end -->

## GOPATH

This project should be cloned to the directory `$GOPATH/src/github.com/agonio/agon`
for when you are developing locally, and require package resolution in your IDE.

This is not required if you are simply building using the `make` targets

## Testing and Building
Make sure you are in the `build` directory to start.

First, let's test all the code. To do this, run `make test`, which will execute all the unit tests for the codebase. 

If you haven't run any of the `build` make targets before then this will also create the Docker based build image,
and then run the tests.

Building the build image may take a few minutes to download all the dependencies, so feel 
free to make cup of tea or coffee at this point. ☕️ 

The build image is only created the first time one of the make targets is executed, and will only rebuild if the build
Dockerfile has changed.

Assuming that the tests all pass, let's go ahead an compile the code and build the Docker images that Agon consists of.

Let's compile and build everything, by running `make build`, this will:

- Compile the Agon Kubernetes integration code
- Create the Docker images that we will later push
- Build the local development tooling for all supported OS's
- Compile and archive the SDKs in various languages

You may note that docker images, and tar archives are tagged with a concatenation of the 
upcoming release number and short git hash for the current commit. This has also been set in 
the code itself, so that it can be seen in via log statements.

Congratulations! You have now successfully tested and built Agon!

### Running a Test Google Kubernetes Engine Cluster

This will setup a test GKE cluster on Google Cloud, with firewall rules set each of the nodes for ports 7000-8000
to be open to UDP traffic.

First step is to create a Google Cloud Project at https://console.cloud.google.com or reuse an existing one.

The build tools (by default) maintain configuration for gcloud and kubectl within the `build` folder, so as to keep
everything seperate (see below for overwriting these config locations). Therefore, once the project has been created,
we will need to authenticate out gcloud tooling against it. To do that run `make gcloud-init` and fill in the
prompts as directed.

Once authenticated, to create the test cluster, run `make gcloud-test-cluster`, which will use the deployment template
found in the `gke-test-cluster` directory. If you would like to change the region and zone the cluster is in, feel free
to edit the `deployment.yaml` file before running this command.  This will take several minutes to complete, but once
done you can go to the Google Cloud Platform console and see that a cluster is up and running! If you want to change the
name of the test cluster you can set the `CLUSTER_NAME` environemnt varlable to value you would like.

To grab the kubectl authentication details for this cluster, run `make gcloud-auth-cluster`, which will generate the
required Kubernetes security credintials for `kubectl`. This will be stored in `build/.kube` by default, but can also be
overwritten by setting the `KUBECONFIG` environment variable before running the command.

Great! Now we are setup, let's try out the development shell, and see if our `kubectl` is working!

Run `make shell` to enter the development shell. You should see a bash shell that has you as the root user.
Enter `kubectl get pods` and press enter. You should see that you have no resources currently, but otherwise see no errors.
Assuming that all works, let's exit the shell by typing `exit` and hitting enter, and look at building, pushing and 
installing Agon next.

To prepare building and pushing images, let's set the REGISTRY environment variable to point to our new project.
You can [choose any registry region](https://cloud.google.com/container-registry/docs/pushing-and-pulling#choosing_a_registry_name)
but for this example, we'll just use `gcr.io`. 

In your shell, run `export REGISTRY=gcr.io/<YOUR-PROJECT-ID>` which will overwrite the default registry settings in our
Make targets. Then, to rebuild our images for this registry, we run `make build` again.

Before we can push the images, there is one more small step! So that we can run regular `docker push` commands 
(rather than `gcloud docker -- push`), we have to authenticate against the registry, which will give us a short
lived key for our local docker config. To do this, run `make gcloud-auth-docker`, and now we have the short lived tokens.

To push our images up at this point, is simple `make push` and that will push up all images you just built to your
project's container registry.

Now that the images are pushed, to install the development version (with all imagePolicies set to always download),
run `make install` and agon will install the image that you just built and pushed on the test cluster you
created at the beginning of this section. (if you want to see the resulting installation yaml, you can find it in `build/.install.yaml`)

### Running a Test Minikube cluster
(Coming soon: Track [this bug](https://github.com/googleprivate/agon/issues/30) for details)

### Next Steps

Have a look in the [examples](../examples) folder to see examples of running Game Servers on Agon.

## Make Variable Reference

### VERSION
The version of this build. Version defaults to the short hash of the latest commit

### REGISTRY
The registry that is being used to store docker images. Defaults to gcr.io/agon-images - the release + CI registry.

### KUBECONFIG
Where the kubectl configuration files are being stored for shell and kubectl targets. Defaults to build/.kube

### CLUSTER_NAME
The (gcloud) test cluster that is being worked against. Defaults to `test-cluster`

## Make Target Reference

All targets will create the build image if it is not present.

### Development Targets

Targets for developing with the build image

#### `make build`
Build all the images required for Agon, as well as the SDKs

#### `make build-images`
Build all the images required for Agon

#### `make build-sdks`
Build all the sdks required for Agon

#### `make build-sdk-cpp`
Build the cpp sdk static and dynamic libraries (linux libraries only)

#### `make test`
Run all tests

#### `make push`
Pushes all built images up to the `$(REGISTRY)`

#### `make shell`
Run a bash shell with the developer tools (go tooling, kubectl, etc) and source code in it.

#### `make godoc`
Run a container with godoc (search index enabled)

#### `make build-gameservers-controller-image`
Compile the gameserver controller and then build the docker image

#### `make build-gameservers-sidecar-image`
Compile the gameserver sidecar and then build the docker image

#### `make gen-crd-client`
Generate the Custom Resource Definition client(s)

#### `make gen-gameservers-sdk-grpc`
Generate the SDK gRPC server and client code

### Build Image Targets

Targets for building the build image

#### `make clean-config`
Cleans the kubernetes and gcloud configurations

#### `make clean-build-image`
Deletes the local build docker image

#### `make build-build-image`
Creates the build docker image

### Google Cloud Platform

A set of utilities for setting up a Container Engine cluster on Google Cloud Platform,
since it's an easy way to get a test cluster working with Kubernetes.

#### `make gcloud-init`
Initialise the gcloud login and project configuration, if you are working with GCP

#### `make gcloud-test-cluster`
Creates and authenticates a small, 3 node GKE cluster to work against

#### `make gcloud-auth-cluster`
Pulls down authentication information for kubectl against a cluster, name can be specified through CLUSTER_NAME
(defaults to 'test-cluster')

#### `make gcloud-auth-docker`
Creates a short lived access to Google Cloud container repositories, so that you are able to call
`docker push` directly. Useful when used in combination with `make push` command.
