# Developing, Testing and Building Agones

Tooling for building and developing against Agones, with only dependencies being
[Make](https://www.gnu.org/software/make/) and [Docker](https://www.docker.com)

Rather than installing all the dependencies locally, you can test and build Agones using the Docker image that is
built from the Dockerfile in this directory. There is an accompanying Makefile for all the common
tasks you may wish to accomplish.

Table of Contents
=================

  * [Building on Different Platforms](#building-on-different-platforms)
     * [Linux](#linux)
     * [Windows](#windows)
     * [macOS](#macos)
  * [GOPATH](#gopath)
  * [Testing and Building](#testing-and-building)
     * [Running a Test Google Kubernetes Engine Cluster](#running-a-test-google-kubernetes-engine-cluster)
     * [Running a Test Minikube cluster](#running-a-test-minikube-cluster)
     * [Running a Test Kind cluster](#running-a-test-kind-cluster)
     * [Running a Custom Test Environment](#running-a-custom-test-environment)
     * [Next Steps](#next-steps)
  * [Make Variable Reference](#make-variable-reference)
     * [VERSION](#version)
     * [REGISTRY](#registry)
     * [KUBECONFIG](#kubeconfig)
     * [CLUSTER_NAME](#cluster_name)
     * [IMAGE_PULL_SECRET](#image_pull_secret)
     * [IMAGE_PULL_SECRET_FILE](#image_pull_secret_file)
  * [Make Target Reference](#make-target-reference)
     * [Development Targets](#development-targets)
        * [make build](#make-build)
        * [make build-images](#make-build-images)
        * [make build-sdks](#make-build-sdks)
        * [make build-sdk](#make-build-sdk)
        * [make run-sdk-conformance-tests](#make-run-sdk-conformance-tests)
        * [make clean-sdk-conformance-tests](#make-clean-sdk-conformance-tests)
        * [make test](#make-test)
        * [make push](#make-push)
        * [make install](#make-install)
        * [make uninstall](#make-uninstall)
        * [make test-e2e](#make-test-e2e)
        * [make test-e2e-integration](#make-test-e2e-integration)
        * [make test-e2e-failure](#make-test-e2e-failure)
        * [make setup-prometheus](#make-setup-prometheus)
        * [make setup-grafana](#make-setup-grafana)
        * [make prometheus-portforward](#make-prometheus-portforward)
        * [make grafana-portforward](#make-grafana-portforward)
        * [make controller-portforward](#make-controller-portforward)
        * [make pprof-web](#make-pprof-web)
        * [make shell](#make-shell)
        * [make godoc](#make-godoc)
        * [make build-controller-image](#make-build-controller-image)
        * [make build-agones-sdk-image](#make-build-agones-sdk-image)
        * [make gen-install](#make-gen-install)
        * [make gen-crd-client](#make-gen-crd-client)
        * [make gen-sdk-grpc](#make-gen-sdk-grpc)
     * [Build Image Targets](#build-image-targets)
        * [make clean-config](#make-clean-config)
        * [make clean-build-image](#make-clean-build-image)
        * [make build-build-image](#make-build-build-image)
     * [Google Cloud Platform](#google-cloud-platform)
        * [make gcloud-init](#make-gcloud-init)
        * [make gcloud-test-cluster](#make-gcloud-test-cluster)
        * [make clean-gcloud-test-cluster](#make-clean-gcloud-test-cluster)
        * [make gcloud-auth-cluster](#make-gcloud-auth-cluster)
        * [make gcloud-auth-docker](#make-gcloud-auth-docker)
     * [Minikube](#minikube)
        * [make minikube-test-cluster](#make-minikube-test-cluster)
        * [make minikube-push](#make-minikube-push)
        * [make minikube-install](#make-minikube-install)
        * [make minikube-setup-prometheus](#make-minikube-setup-prometheus)
        * [make minikube-setup-grafana](#make-minikube-setup-grafana)
        * [make minikube-prometheus-portforward](#make-minikube-prometheus-portforward)
        * [make minikube-grafana-portforward](#make-minikube-grafana-portforward)
        * [make minikube-test-e2e](#make-minikube-test-e2e)
        * [make minikube-shell](#make-minikube-shell)
        * [make minikube-transfer-image](#make-minikube-transfer-image)
        * [make minikube-controller-portforward](#make-minikube-controller-portforward)
     * [Kind](#Kind)
        * [make kind-test-cluster](#make-kind-test-cluster)
        * [make kind-push](#make-kind-push)
        * [make kind-install](#make-kind-install)
        * [make kind-setup-prometheus](#make-kind-setup-prometheus)
        * [make kind-setup-grafana](#make-kind-setup-grafana)
        * [make kind-prometheus-portforward](#make-kind-prometheus-portforward)
        * [make kind-grafana-portforward](#make-kind-grafana-portforward)
        * [make kind-test-e2e](#make-kind-test-e2e)
        * [make kind-shell](#make-kind-shell)
        * [make kind-controller-portforward](#make-kind-controller-portforward)
  * [Dependencies](#dependencies)
  * [Troubleshooting](#troubleshooting)
      * [$GOPATH/$GOROOT error when building in WSL](#gopathgoroot-error-when-building-in-wsl)
      * [Error: cluster-admin-binding already exists](#error-cluster-admin-binding-already-exists)
      * [Error: releases do not exist](#error-releases-do-not-exist)
      * [I want to use pprof to profile the controller](#i-want-to-use-pprof-to-profile-the-controller)

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

First, let's test the Agones system code. To do this, run `make test-go`, which will execute all the unit tests for
the Go codebase (there are other tests, but they can take a long time to run).

If you haven't run any of Make targets before then this will also create the Docker based build
image, and then run the tests.

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

If you don't have a long time to kill, you can run `make build-images` to only build the images for running Agones
, which is often all you need for development.

Congratulations! You have now successfully tested and built Agones!

### Running a Test Google Kubernetes Engine Cluster

This will setup a test GKE cluster on Google Cloud, with firewall rules set for each of the nodes for ports 7000-8000
to be open to UDP traffic.

First step is to create a Google Cloud Project at https://console.cloud.google.com or reuse an existing one.

The build tools (by default) maintain configuration for gcloud within the `build` folder, so as to keep
everything separate (see below for overwriting these config locations). Therefore, once the project has been created,
we will need to authenticate out gcloud tooling against it. To do that run `make gcloud-init` and fill in the
prompts as directed.

Once authenticated, to create the test cluster, run `make gcloud-test-cluster`, which will use the Terraform
configuration found in the `build/terraform/gke` directory.

You can customize GKE cluster via environment variables or by using a [`local-includes`](./local-includes) file.
See the table below for available customizations :

| Parameter                             | Description                                                                   | Default       |
|---------------------------------------|-------------------------------------------------------------------------------|---------------|
| `GCP_CLUSTER_NAME`                    | The name of the cluster                                                       | `test-cluster`  |
| `GCP_CLUSTER_ZONE`                    | The name of the Google Compute Engine zone in which the cluster will resides. |  `us-west1-c`   |
| `GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT`| The number of nodes to create in this cluster.                                |  `4`            |
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

When your are finished, you can run `make clean-gcloud-test-cluster` to tear down your cluster.

### Running a Test Minikube cluster
This will setup a [Minikube](https://github.com/kubernetes/minikube) cluster, running on an `agones` profile, 

Because Minikube runs on a virtualisation layer on the host, some of the standard build and development Make targets
need to be replaced by Minikube specific targets.

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

Running end-to-end tests on Minikube is done via the `make minikube-test-e2e` target. This target use the same `make test-e2e` but also setup some prerequisites for use with a Minikube cluster.

### Running a Test Kind cluster
 This will setup a [Kubernetes IN Docker](https://github.com/kubernetes-sigs/kind) cluster named agones by default.

Because Kind runs on a docker on the host, some of the standard build and development Make targets
need to be replaced by kind specific targets.

First, [install Kind](https://github.com/kubernetes-sigs/kind#installation-and-usage).

Next we will create the Agones Kind cluster. Run `make kind-test-cluster` to create the `agones` Kubernetes cluster.

This will also setup helm and a kubeconfig, the kubeconfig location can found using the following command `kind get kubeconfig-path --name=agones` assuming you're using the default `KIND_PROFILE`.

You can verify that your new cluster information by running (if you don't have kubectl you can skip to the shell section):

```
KUBECONFIG=$(kind get kubeconfig-path --name=agones) kubectl cluster-info
```

Great! Now we are setup, we also provide a development shell with a handful set of tools like kubectl and helm.

Run `make kind-shell` to enter the development shell. You should see a bash shell that has you as the root user.
Enter `kubectl get pods` and press enter. You should see that you have no resources currently, but otherwise see no errors.
Assuming that all works, let's exit the shell by typing `exit` and hitting enter, and look at a building, pushing and installing Agones on Kind next.

You may remember in the first part of this walkthrough, we ran `make build`, which created all the images and binaries
we needed to work with Agones locally. We can push these images them straight into kind very easily!

Run `make kind-push` which will send all of Agones's docker images from your local Docker into the Agones Kind container.

Now that the images are pushed, to install the development version,
run `make kind-install` and Agones will install the images that you built and pushed to the Agones Kind cluster.

Running end-to-end tests on Kind is done via the `make kind-test-e2e` target. This target use the same `make test-e2e` but also setup some prerequisites for use with a Kind cluster.

If you are having performance issues, check out these docs [here](https://github.com/kubernetes-sigs/kind/tree/master/docs/user#creating-a-cluster)

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
   
Now you're ready to begin the development/test cycle:
- `make build` will build Agones
- `make test` will run local tests, which includes `site-test` target
- `make push` will push the Agones images to your image repository 
- `make install` will install/upgrade Agones into your cluster
- `make test-e2e` will run end-to-end tests in your cluster

You can combine some of the above steps into a single one, for example `make build push install` or `make build push test-e2e`.

If you need to clean-up your cluster, you can use `make uninstall` to remove Agones and `make clean-custom-test-cluster` to reset helm.

### Next Steps

Have a look in the [examples](../examples) folder to see examples of running Game Servers on Agones.

## Make Variable Reference

### VERSION
The version of this build. Version defaults to the short hash of the latest commit.

### REGISTRY
The registry that is being used to store docker images. Defaults to gcr.io/agones-images - the release + CI registry.

### KUBECONFIG
The Kubernetes config file used to access the cluster. Defaults to `~/.kube/config` - the file used by default by kubectl.

### CLUSTER_NAME
The (gcloud) test cluster that is being worked against. Defaults to `test-cluster`.

### GCP_PROJECT
Your GCP project for deploying GKE cluster. Defaults to gcloud default project settings.

### GKE_PASSWORD
If specified basic authentication would be enabled for your cluster with username "admin".
Empty string `""` would disable basic authentication.

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

Targets for developing with the build image.

#### `make build`
Build all the images required for Agones, as well as the SDKs

#### `make build-images`
Build all the images required for Agones

#### `make build-sdks`
Build all the sdks required for Agones

#### `make build-sdk`
Next command `make build-sdk SDK_FOLDER=[SDK_TYPE]` will build SDK of `SDK_TYPE`. 
For instance, in order to build the cpp sdk static and dynamic libraries (linux libraries only) use `SDK_FOLDER=cpp`

#### `make run-sdk-conformance-local`
Run Agones sidecar which would wait for all requests from the SDK client.
Note that annotation should contain UID and label should contain CreationTimestamp values to pass the test.

#### `make run-sdk-conformance-no-build`
Only run a conformance test for a specific Agones SDK.

#### `make run-sdk-conformance-test`
Build, run and clean conformance test for a specific Agones SDK.

#### `make run-sdk-conformance-tests`
Run SDK conformance test.
Run SDK server (sidecar) in test mode (which would record all GRPC requests) versus all SDK test clients which should generate those requests. All methods are verified.

#### `make clean-sdk-conformance-tests`
Clean leftover binary and package files after running SDK conformance tests.

#### `make test`
Run the linter and tests

#### `make build-examples`
Run `make build` for all `examples` subdirectories

#### `make site-server`
Generate `https://agones.dev` website locally and host on `http://localhost:1313`

#### `make hugo-test`
Check the links in a website

#### `make site-test`
Check the links in a website, includes `test-gen-api-docs` target

#### `make site-images`
Create all the site images from dot and puml diagrams in /site/static/diagrams

#### `make gen-api-docs`
Generate Agones CRD reference documentation [Agones CRD API reference](../site/content/en/docs/Reference/agones_crd_api_reference.html). Set `feature` shortcode with proper version automatically

#### `make test-gen-api-docs`
Verifies that there is no changes in generated [Agones CRD API reference](../site/content/en/docs/Reference/agones_crd_api_reference.html) compared to the current one (useful for CI)

#### `make push`
Pushes all built images up to the `$(REGISTRY)`

#### `make install`
Installs the current development version of Agones into the Kubernetes cluster

#### `make uninstall`
Removes Agones from the Kubernetes cluster

#### `make test-e2e`
Runs end-to-end tests on the previously installed version of Agones.
These tests validate Agones flow from start to finish.

It uses the KUBECONFIG to target a Kubernetes cluster.

Use `GAMESERVERS_NAMESPACE` flag to provide a namespace or leave it empty in order to create and use a random one.

See [`make minikube-test-e2e`](#make-minikube-test-e2e) to run end-to-end tests on Minikube.

#### `make test-e2e-integration`
Runs integration portion of the end-to-end tests.

Pass flags to [go test](https://golang.org/cmd/go/#hdr-Testing_flags) command
using the `ARGS` parameter. For example, to run only the `TestGameServerReserve` test:

```bash
make test-e2e-integration ARGS='-run TestGameServerReserve'
```

#### `make test-e2e-failure`
Run controller failure portion of the end-to-end tests.

#### `make setup-prometheus`

Install Prometheus server using [stable/prometheus](https://github.com/helm/charts/tree/master/stable/prometheus) chart into the current cluster.

By default all exporters and alertmanager is disabled.

You can use this to collect Agones [Metrics](../site/content/en/docs/Guides/metrics.md).

See [`make minikube-setup-prometheus`](#make-minikube-setup-prometheus) and [`make kind-setup-prometheus`](#make-kind-setup-prometheus) to run the installation on Minikube or Kind.

#### make helm-repo-update

Run helm repo update to get the mose recent charts.

#### `make setup-grafana`

Install Gafrana server using [stable/grafana](https://github.com/helm/charts/tree/master/stable/grafana) chart into the current cluster and setup [Agones dashboards with Prometheus datasource](./grafana/).

You can set your own password using the `PASSWORD` environment variable.

See [`make minikube-setup-grafana`](#make-minikube-setup-grafana) and [`make kind-setup-grafana`](#make-kind-setup-grafana) to run the installation on Minikube or Kind.

#### `make prometheus-portforward`

Sets up port forwarding to the
Prometheus deployment (port 9090 Prometheus UI).

On Windows and MacOS you will need to have installed [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

See [`make minikube-prometheus-portforward`](#make-minikube-prometheus-portforward) and [`make kind-prometheus-portforward`](#make-minikube-prometheus-portforward) to run  on Minikube or Kind.

#### `make grafana-portforward`

Sets up port forwarding to the
grafana deployment (port 3000 UI).

On Windows and MacOS you will need to have installed [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

See [`make minikube-grafana-portforward`](#make-minikube-grafana-portforward) and [`make kind-grafana-portforward`](#make-minikube-grafana-portforward) to run  on Minikube or Kind.

#### `make controller-portforward`

Sets up port forwarding to a specified PORT var (defaults to 8080 for controller metrics) to the
controller deployment.

#### `make pprof-web`

Start the web interface for pprof.

#### `make shell`
Run a bash shell with the developer tools (go tooling, kubectl, etc) and source code in it.

#### `make godoc`
Run a container with godoc (search index enabled)

#### `make build-controller-image`
Compile the gameserver controller and then build the docker image

#### `make build-agones-sdk-image`
Compile the gameserver sidecar and then build the docker image

#### `make build-ping-image`
Compile the ping binary and then build the docker image

#### `make gen-install`
Generate the `/install/yaml/install.yaml` from the Helm template

#### `make gen-crd-client`
Generate the Custom Resource Definition client(s)

#### `make gen-sdk-grpc`
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
Initialise the gcloud login and project configuration, if you are working with GCP.

#### `make gcloud-test-cluster`
Creates and authenticates a GKE cluster to work against.

#### `make clean-gcloud-test-cluster`
Delete a GKE cluster previously created with `make gcloud-test-cluster`.

#### `make gcloud-auth-cluster`
Pulls down authentication information for kubectl against a cluster, name can be specified through CLUSTER_NAME
(defaults to 'test-cluster').

#### `make gcloud-auth-docker`
Creates a short lived access to Google Cloud container repositories, so that you are able to call
`docker push` directly. Useful when used in combination with `make push` command.

### Terraform

Utilities for deploying a Kubernetes Engine cluster on Google Cloud Platform using `google` Terraform provider.

#### `make gcloud-terraform-cluster`
Create GKE cluster and install release version of agones.
Run next command to create GKE cluster with agones (version from helm repository):
```
[GKE_PASSWORD="<YOUR_PASSWORD>"] make gcloud-terraform-cluster
```
Where `<YOUR_PASSWORD>` should be at least 16 characters in length. You can omit GKE_PASSWORD and then basic auth would be disabled. Also you change `ports="7000-8000"` setting using tfvars file.
Also you can define password `password=<YOUR_PASSWORD>` string in `build/terraform.tfvars`.
Change AGONES_VERSION to a specific version you want to install.

#### `make gcloud-terraform-install`
Create GKE cluster and install current version of agones.
The current version should be built and pushed to release_registry beforehand:
```
make build-images
make push
```

#### `make gcloud-terraform-destroy-cluster`
Run `terraform destroy` on your cluster.

#### `make terraform-clean`
Remove .terraform directory with configs as well as tfstate files.

#### `make terraform-test GCP_PROJECT="<YOUR_PROJECT_ID>"`
Run Golang test which emulates and verifies successful execution of next two steps:
```
make gcloud-terraform-cluster
make gcloud-terraform-destroy-cluster
```
Singleton, could not be executed in parallel with itself. As it uses the one terraform tfstate file.

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

#### `make minikube-setup-prometheus`
Installs prometheus metric backend into the Kubernetes cluster.
Use this instead of `make setup-prometheus`, as it disables Persistent Volume Claim.

#### `make minikube-setup-grafana`

Installs grafana into the Kubernetes cluster.
Use this instead of `make setup-grafana`, as it disables Persistent Volume Claim.

#### `make minikube-prometheus-portforward`

The minikube version of [`make prometheus-portforward`](#make-prometheus-portforward) to setup
port forwarding to the prometheus deployment.  

#### `make minikube-grafana-portforward`

The minikube version of [`make grafana-portforward`](#make-grafana-portforward) to setup
port forwarding to the grafana deployment.  

#### `make minikube-test-e2e`
Runs end-to-end tests on the previously installed version of Agones.
These tests validate Agones flow from start to finish.

#### `make minikube-shell`
Connecting to Minikube requires so enhanced permissions, so use this target
instead of `make shell` to start an interactive shell for development on Minikube.

#### `make minikube-transfer-image`
Convenience target for transferring images into minikube.
Use TAG to specify the image to transfer into minikube

#### `make minikube-controller-portforward`
The minikube version of [`make controller-portforward`](#make-controller-portforward) to setup
port forwarding to the controller deployment.

### Kind

[Kind - kubernetes in docker](https://github.com/kubernetes-sigs/kind) is a tool for running local Kubernetes clusters using Docker container "nodes".
Kind is primarily designed for testing Kubernetes 1.11+, initially targeting the [conformance tests](https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md).

Since Kind runs locally, there are some targets that need to be used instead of the standard ones above.

#### `make kind-test-cluster`
Starts a local kubernetes cluster, you can delete it with `make kind-delete-cluster`

Use KIND_PROFILE variable to change the name of the cluster.

#### `make kind-push`
Push the local Agones Docker images that have already been built 
via `make build` or `make build-images` into the "agones" Kind cluster.

#### `make kind-install`
Installs the current development version of Agones into the Kubernetes cluster.
Use this instead of `make install`, as it disables PullAlways on the install.yaml

#### `make kind-setup-prometheus`

Installs prometheus metric backend into the Kubernetes cluster.
Use this instead of `make setup-prometheus`, as it disables Persistent Volume Claim.

#### `make kind-setup-grafana`

Installs grafana into the Kubernetes cluster.
Use this instead of `make setup-grafana`, as it disables Persistent Volume Claim.

#### `make kind-prometheus-portforward`

The minikube version of [`make prometheus-portforward`](#make-prometheus-portforward) to setup
port forwarding to the prometheus deployment.  

#### `make kind-grafana-portforward`

The minikube version of [`make grafana-portforward`](#make-grafana-portforward) to setup
port forwarding to the grafana deployment.  


#### `make kind-test-e2e`
Runs end-to-end tests on the previously installed version of Agones.
These tests validate Agones flow from start to finish.

#### `make kind-shell`
Connecting to Kind requires so enhanced permissions, so use this target
instead of `make shell` to start an interactive shell for development on Kind.

#### `make kind-controller-portforward`
The Kind version of [`make controller-portforward`](#make-controller-portforward) to setup
port forwarding to the controller deployment.

## Dependencies

This project uses the [go modules](https://github.com/golang/go/wiki/Modules) as its manager. You can see the list of dependencies [here](https://github.com/googleforgames/agones/blob/master/go.mod).

#### Vendoring

Agones uses [module vendoring](https://tip.golang.org/cmd/go/#hdr-Modules_and_vendoring) to reliably produce versioned builds with consistent behavior.

Adding a new dependency to Agones:

*  `go mod tidy` This will import your new deps into the go.mod file and trim out any removed dependencies.
*  `go mod vendor` Pulls module code into the vendor directory.

Sometimes the code added to vendor may not include a subdirectory that houses code being used but not as an import
(protos passed as args to scripts is a good example). In this case you can go into the module cache and copy what you need to the path in vendor. 

Here is an example for getting third_party from grpc-ecosystem/grpc-gateway v1.5.1 into vendor:

*  AGONES_PATH=/wherever/your/agones/path/is
*  cp -R $GOPATH/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.5.1/third_party $AGONES_PATH/vendor/github.com/grpc-ecosystem/grpc-gateway/

Note the version in the pathname. Go may eliminate the need to do this in future versions.

We also use vendor to hold code patches while waiting for the project to release the fixes in their own code. An example is in [k8s.io/apimachinery](https://github.com/googleforgames/agones/issues/414) where a fix will be released later this year, but we updated our own vendored version in order to fix the issue sooner.


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

#### Error: cluster-admin-binding already exists

This surfaces while running `make gcloud-auth-cluster`. The solution is to run `kubectl describe clusterrolebinding | grep cluster-admin-binding- -A10`, find clusterrolebinding which belongs to your `User` account and then run `kubectl delete clusterrolebindings cluster-admin-binding-<md5Hash>` where `<md5Hash>` is a value specific to your account. Now you can execute `make gcloud-auth-cluster` again. If you run into a permission denied error when attempting the delete operation, you need to run `sudo chown <your username> <path to .kube/config>` to change ownership of the file to yourself.

#### Error: releases do not exist

Run `make uninstall` then run `make install` again.

#### I want to use pprof to profile the controller.

Run `make build-images GO_BUILD_TAGS=profile` and this will build images with [pprof](https://golang.org/pkg/net/http/pprof/)
enabled in the controller, which you can then push and install on your cluster.

To get the pprof ui working, run `make controller-portforward PORT=6060` (or `minikube-controller-portforward PORT=6060` if you are on minikube),
which will setup the port forwarding to the pprof http endpoint.

Run `make pprof-web`, which will start the web interface. It may take a few minutes to start up, but it can be opened on
[http://localhost:6060/ui](http://localhost:6060/ui).

