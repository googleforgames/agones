# Developing, Testing and Building Agones

Tooling for building and developing against Agones, with only dependencies being
[Make](https://www.gnu.org/software/make/) and [Docker](https://www.docker.com)

Rather than installing all the dependencies locally, you can test and build Agones using the Docker image that is
built from the Dockerfile in this directory. There is an accompanying Makefile for all the common
tasks you may wish to accomplish.

<!-- ToC start -->
# Table of Contents

   1. [Table of Contents](#table-of-contents)
   1. [Building on Different Platforms](#building-on-different-platforms)
      1. [Linux](#linux)
      1. [Windows](#windows)
      1. [macOS](#macOS)
   1. [GOPATH](#gopath)
   1. [Testing and Building](#testing-and-building)
      1. [Running a Test Google Kubernetes Engine Cluster](#running-a-test-google-kubernetes-engine-cluster)
      1. [Running a Test Minikube cluster](#running-a-test-minikube-cluster)
      1. [Running a Custom Test Environment](#running-a-custom-test-environment)
      1. [Next Steps](#next-steps)
   1. [Make Variable Reference](#make-variable-reference)
      1. [VERSION](#version)
      1. [REGISTRY](#registry)
      1. [KUBECONFIG](#kubeconfig)
      1. [CLUSTER_NAME](#cluster_name)
      1. [IMAGE_PULL_SECRET](#image_pull_secret)
      1. [IMAGE_PULL_SECRET_FILE](#image_pull_secret_file)
   1. [Make Target Reference](#make-target-reference)
      1. [Development Targets](#development-targets)
      1. [Build Image Targets](#build-image-targets)
      1. [Google Cloud Platform](#google-cloud-platform)
      1. [Minikube](#minikube)
      1. [Custom Environment](#custom-environment)
   1. [Dependencies](#dependencies)
   1. [Troubleshooting](#troubleshooting)
   
<!-- ToC end -->

## Building on Different Platforms

### Linux
- Install Make, either via `apt install make` or `yum install make` depending on platform.
- [Install Docker](https://docs.docker.com/engine/installation/) for your Linux platform.
- (optional) Minikube will require [VirtualBox](https://www.virtualbox.org) and will need to be installed if you wish
  to develop on Minikube 

### Windows
Building and developing Agones requires you to use the 
[Windows Subsystem for Linux](https://blogs.msdn.microsoft.com/wsl/)(WSL),
as this makes it easy to create a (relatively) cross platform development and build system.

- [Install WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10)
  - Preferred release is [Ubuntu 16.04](https://www.microsoft.com/en-us/store/p/ubuntu/9nblggh4msv6) or greater.
- [Install Docker for Windows](https://docs.docker.com/docker-for-windows/install/)
- Within WSL, Install [Docker for Ubuntu](https://docs.docker.com/engine/installation/linux/docker-ce/ubuntu/)
- Follow [this guide](https://nickjanetakis.com/blog/setting-up-docker-for-windows-and-wsl-to-work-flawlessly)
  from "Configure WSL to Connect to Docker for Windows" forward 
  for integrating the Docker on WSL with the Windows Docker installation
  - Note the binding of `/c` to `/mnt/c` (or drive of your choice) - this is very important!
- Agones will need to be cloned somewhere on your `/c` (or drive of your choice) path, as that is what Docker will support mounts from
- All interaction with Agones must be on the `/c` (or drive of your choice) path, otherwise Docker mounts will not work
- Now the `make` commands can all be run from within your WSL shell
- (optional) Minikube is supported via the [HyperV](https://docs.microsoft.com/en-us/virtualization/hyper-v-on-windows/index) 
  driver - the same virtualisation platform as the Docker installation.
- **Note**: If you want to dev and test with Minikube, you **must** run WSL as Administrator, otherwise Minikube can't control HyperV.

### macOS

- Install Make, `brew install make`, if it's not installed already
- Install [Docker for Mac](https://docs.docker.com/docker-for-mac/install/)
- (optional) Minikube will require [VirtualBox](https://www.virtualbox.org) and will need to be installed if you wish
  to develop on Minikube 

## GOPATH

This project should be cloned to the directory `$GOPATH/src/agones.dev/agones`
for when you are developing locally, and require package resolution in your IDE.

If you have a working [Go environment](https://golang.org/doc/install), you can also do this through:

```bash
go get -d agones.dev/agones
cd $GOPATH/src/agones.dev/agones
```

This is not required if you are simply building using the `make` targets, and do not plan to edit the code 
in an IDE.

If you are not familiar with GOPATHs, you can read [How to Write Go Code](https://golang.org/doc/code.html). 

## Testing and Building

Make sure you are in the `build` directory to start.

First, let's test all the code. To do this, run `make test`, which will execute all the unit tests for the codebase. 

If you haven't run any of the `build` make targets before then this will also create the Docker based build image,
and then run the tests.

Building the build image may take a few minutes to download all the dependencies, so feel 
free to make cup of tea or coffee at this point. ☕️ 

**Note**: If you get build errors and you followed all the instructions so far, consult the [Troubleshooting](#troubleshooting) section

The build image is only created the first time one of the make targets is executed, and will only rebuild if the build
Dockerfile has changed.

Assuming that the tests all pass, let's go ahead an compile the code and build the Docker images that Agones consists of.

Let's compile and build everything, by running `make build`, this will:

- Compile the Agones Kubernetes integration code
- Create the Docker images that we will later push
- Build the local development tooling for all supported OS's
- Compile and archive the SDKs in various languages

You may note that docker images, and tar archives are tagged with a concatenation of the 
upcoming release number and short git hash for the current commit. This has also been set in 
the code itself, so that it can be seen in via log statements.

Congratulations! You have now successfully tested and built Agones!

### Running a Test Google Kubernetes Engine Cluster

This will setup a test GKE cluster on Google Cloud, with firewall rules set for each of the nodes for ports 7000-8000
to be open to UDP traffic.

First step is to create a Google Cloud Project at https://console.cloud.google.com or reuse an existing one.

The build tools (by default) maintain configuration for gcloud within the `build` folder, so as to keep
everything separate (see below for overwriting these config locations). Therefore, once the project has been created,
we will need to authenticate out gcloud tooling against it. To do that run `make gcloud-init` and fill in the
prompts as directed.

Once authenticated, to create the test cluster, run `make gcloud-test-cluster`, which will use the deployment template
found in the `gke-test-cluster` directory.

You can customize GKE cluster via environment variables or by using a [`local-includes`](./local-includes) file.
See the table below for available customizations :

| Parameter                             | Description                                                                   | Default       |
|---------------------------------------|-------------------------------------------------------------------------------|---------------|
| `GCP_CLUSTER_NAME`                    | The name of the cluster                                                       | `test-cluster`  |
| `GCP_CLUSTER_ZONE`                    | The name of the Google Compute Engine zone in which the cluster will resides. |  `us-west1-c`   |
| `GCP_CLUSTER_LEGACYABAC`              | Enables or disables the [ABAC](https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.zones.clusters#LegacyAbac) authorization mechanism on a cluster.            | `false`         |
| `GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT`| The number of nodes to create in this cluster.                                |  `3`            |
| `GCP_CLUSTER_NODEPOOL_MACHINETYPE`    | The name of a Google Compute Engine machine type.                             | `n1-standard-4` |

If you would like to change more settings, feel free to edit the [`cluster.yml.jinja`](./gke-test-cluster/cluster.yml.jinja) file before running this command.  

This will take several minutes to complete, but once done you can go to the Google Cloud Platform console and see that 
a cluster is up and running!

To grab the kubectl authentication details for this cluster, run `make gcloud-auth-cluster`, which will generate the
required Kubernetes security credentials for `kubectl`. This will be stored in `~/.kube/config` by default, but can also be
overwritten by setting the `KUBECONFIG` environment variable before running the command.

Great! Now we are setup, let's try out the development shell, and see if our `kubectl` is working!

Run `make shell` to enter the development shell. You should see a bash shell that has you as the root user.
Enter `kubectl get pods` and press enter. You should see that you have no resources currently, but otherwise see no errors.
Assuming that all works, let's exit the shell by typing `exit` and hitting enter, and look at building, pushing and 
installing Agones next.

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
run `make install` and Agones will install the image that you just built and pushed on the test cluster you
created at the beginning of this section. (if you want to see the resulting installation yaml, you can find it in `build/.install.yaml`)

Finally to run end-to-end tests against your development version previously installed in your test cluster run `make test-e2e`, this will validate the whole application flow (from start to finish). If you're curious about how they work head to [tests/e2e](../test/e2e/)

### Running a Test Minikube cluster
This will setup a [Minikube](https://github.com/kubernetes/minikube) cluster, running on an `agones` profile, 

Because Minikube runs on a virtualisation layer on the host, some of the standard build and development Make targets
need to be replaced by Minikube specific targets.

> We recommend installing version [0.29.0 of minikube](https://github.com/kubernetes/minikube/releases/tag/v0.29.0).

First, [install Minikube](https://github.com/kubernetes/minikube#installation), which may also require you to install
a virtualisation solution, such as [VirtualBox](https://www.virtualbox.org) as well.
Check the [Building on Different Platforms](#building-on-different-platforms) above for details on what virtualisation 
solution to use.

Next we will create the Agones Minikube cluster. Run `make minikube-test-cluster` to create the `agones` profile,
and a Kubernetes cluster of the supported version under this profile.

This will also install the kubectl authentication credentials in `~/.kube`, and set the 
[`kubectl` context](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/) 
to `agones`.

Great! Now we are setup, let's try out the development shell, and see if our `kubectl` is working!

Run `make minikube-shell` to enter the development shell. You should see a bash shell that has you as the root user.
Enter `kubectl get pods` and press enter. You should see that you have no resources currently, but otherwise see no errors.
Assuming that all works, let's exit the shell by typing `exit` and hitting enter, and look at a building, pushing and 
installing Agones on Minikube next.

You may remember in the first part of this walkthrough, we ran `make build`, which created all the images and binaries
we needed to work with Agones locally. We can push these images them straight into Minikube very easily!

Run `make minikube-push` which will send all of Agones's docker images from your local Docker into the Agones Minikube
instance.

Now that the images are pushed, to install the development version,
run `make minikube-install` and Agones will install the images that you built and pushed to the Agones Minikube instance 
(if you want to see the resulting installation yaml, you can find it in `build/.install.yaml`).

It's worth noting that Minikube does let you [reuse its Docker daemon](https://github.com/kubernetes/minikube/blob/master/docs/reusing_the_docker_daemon.md),
and build directly on Minikube, but in this case this approach is far simpler, 
and makes cross-platform support for the build system much easier.

If you find you also want to push your own images into Minikube, 
the convenience make target `make minikube-transfer-image` can be run with the `TAG` argument specifying 
the tag of the Docker image you wish to transfer into Minikube.

For example:
```bash
$ make minikube-transfer-image TAG=myimage:0.1
```

Running end-to-end tests on minikube is done via the `make minikube-test-e2e` target. This target use the same `make test-e2e` but also setup some prerequisites for use with a minikube cluster.

### Running a Custom Test Environment

This section is addressed to developers using a custom Kubernetes provider, a custom image repository and/or multiple test clusters.

Prerequisites:
- a(some) running k8s cluster(s)
- Have kubeconfig file(s) ready
- docker must be logged into the image repository you're going to use

To begin, you need to set up the following environment variables:
- `KUBECONFIG` should point to the kubeconfig file used to access the cluster; 
   if unset, it defaults to `~/.kube/config`
- `REGISTRY` should point to your image repository of your choice (i.e. gcr.io/<YOUR-PROJECT-ID>)
- `IMAGE_PULL_SECRET` must contain the name of the secret required to pull the Agones images, 
   in case you're using a custom repository; if unset, no pull secret will be used
- `IMAGE_PULL_SECRET_FILE` must be initialized to the full path of the file containing
   the secret for pulling the Agones images, in case of a custom image repository; 
   if set, `make install` will install this secret in both the `agones-system` (for pulling the controller image)
   and `default` (for pulling the sdk image) repositories
   
The second step is to prepare your cluster for the Agones deployments. Run `make setup-custom-test-cluster` to install helm in it.

Now you're ready to begin the development/test cycle:
- `make build` will build Agones
- `make test` will run local tests
- `make push` will push the Agones images to your image repository 
- `make test-e2e` will run end-to-end tests in your cluster
- `make install` will install/upgrade Agones into your cluster

You can combine some of the above steps into a single one, for example `make build push install` or `make build push test-e2e`.

If you need to clean-up your cluster, you can use `make uninstall` to remove Agones and `make clean-custom-test-cluster` to reset helm.

### Next Steps

Have a look in the [examples](../examples) folder to see examples of running Game Servers on Agones.

## Make Variable Reference

### VERSION
The version of this build. Version defaults to the short hash of the latest commit

### REGISTRY
The registry that is being used to store docker images. Defaults to gcr.io/agones-images - the release + CI registry.

### KUBECONFIG
The Kubernetes config file used to access the cluster. Defaults to `~/.kube/config` - the file used by default by kubectl.

### CLUSTER_NAME
The (gcloud) test cluster that is being worked against. Defaults to `test-cluster`

### IMAGE_PULL_SECRET
The name of the secret required to pull the Agones images, if needed.
If unset, no pull secret will be used.

### IMAGE_PULL_SECRET_FILE
The full path of the file containing the secret for pulling the Agones images, in case it's needed.

If set, `make install` will install this secret in both the `agones-system` (for pulling the controller image)
and `default` (for pulling the sdk image) repositories.
   
## Make Target Reference

All targets will create the build image if it is not present.

### Development Targets

Targets for developing with the build image

#### `make build`
Build all the images required for Agones, as well as the SDKs

#### `make build-images`
Build all the images required for Agones

#### `make build-sdks`
Build all the sdks required for Agones

#### `make build-sdk-cpp`
Build the cpp sdk static and dynamic libraries (linux libraries only)

#### `make test`
Run the linter and tests

#### `make push`
Pushes all built images up to the `$(REGISTRY)`

#### `make install`
Installs the current development version of Agones into the Kubernetes cluster

#### `make uninstall`
Removes Agones from the Kubernetes cluster

### `make test-e2e`
Runs end-to-end tests on the previously installed version of Agones.
These tests validate Agones flow from start to finish.

It uses the KUBECONFIG to target a Kubernetes cluster.

See [`make minikube-test-e2e`](#make-minikube-test-e2e) to run end-to-end tests on minikube.

#### `make shell`
Run a bash shell with the developer tools (go tooling, kubectl, etc) and source code in it.

#### `make godoc`
Run a container with godoc (search index enabled)

#### `make build-agones-controller-image`
Compile the gameserver controller and then build the docker image

#### `make build-agones-sdk-image`
Compile the gameserver sidecar and then build the docker image

#### `make gen-install`
Generate the `/install/yaml/install.yaml` from the Helm template

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

A set of utilities for setting up a Kubernetes Engine cluster on Google Cloud Platform,
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

### Minikube

A set of utilities for setting up and running a [Minikube](https://github.com/kubernetes/minikube) instance, 
for local development.

Since Minikube runs locally, there are some targets that need to be used instead of the standard ones above.

#### `make minikube-test-cluster`
Switches to an "agones" profile, and starts a kubernetes cluster
of the right version.

Use MINIKUBE_DRIVER variable to change the VM driver
(defaults virtualbox for Linux and macOS, hyperv for windows) if you so desire.

#### `make minikube-push`
Push the local Agones Docker images that have already been built 
via `make build` or `make build-images` into the "agones" minikube instance.

#### `make minikube-install`
Installs the current development version of Agones into the Kubernetes cluster.
Use this instead of `make install`, as it disables PullAlways on the install.yaml

### `make minikube-test-e2e`
Runs end-to-end tests on the previously installed version of Agones.
These tests validate Agones flow from start to finish.

#### `make minikube-shell`
Connecting to Minikube requires so enhanced permissions, so use this target
instead of `make shell` to start an interactive shell for development on Minikube.

#### `make minikube-transfer-image`
Convenience target for transferring images into minikube.
Use TAG to specify the image to transfer into minikube

### Custom Environment

#### `make setup-custom-test-cluster`
Initializes your custom cluster for working with Agones, by installing Helm/Tiller.

#### `make clean-custom-test-cluster`
Cleans up your custom cluster by reseting Helm.

## Dependencies

This project uses the [dep](https://github.com/golang/dep) as a dependency manager. You can see the list of dependencies [here](https://github.com/GoogleCloudPlatform/agones/blob/master/Gopkg.toml).

## Troubleshooting

Frequent issues and possible solutions

#### $GOPATH/$GOROOT error when building in WSL

If you get this error when building Agones in WSL (`make build`, `make test` or any other related target):

```can't load package: package agones.dev/agones/cmd/controller: cannot find package "agones.dev/agones/cmd/controller" in any of:
       /usr/local/go/src/agones.dev/agones/cmd/controller (from $GOROOT)
       /go/src/agones.dev/agones/cmd/controller (from $GOPATH)
```

- Are your project files on a different folder than C? If yes, then you should either move them on drive C or set up Docker for Windows to share your project drive as well
- Did you set up the volume mount for Docker correctly? By default, drive C is mapped by WSL as /mnt/c, but Docker expects it as /c. You can test by executing `ls /c` in your linux shell. If you get an error, then follow the instructions for [setting up volume mount for Docker](https://nickjanetakis.com/blog/setting-up-docker-for-windows-and-wsl-to-work-flawlessly#ensure-volume-mounts-work)


