# Cluster Setup Guide

This guide covers setting up different types of Kubernetes clusters for Agones development.

## Running a Test Google Kubernetes Engine Cluster

This will setup a test GKE cluster on Google Cloud, with firewall rules set for each of the nodes for ports 7000-8000
to be open to UDP traffic.

First step is to create a Google Cloud Project at https://console.cloud.google.com or reuse an existing one.

The build tools (by default) maintain configuration for gcloud within the `build` folder, to keep
everything separate (see below for overwriting these config locations). Therefore, once the project has been created,
we will need to authenticate our gcloud tooling against it. To do that run `make gcloud-init` and fill in the
prompts as directed.

Once authenticated, to create the test cluster, run `make gcloud-test-cluster`
(or run `make gcloud-e2e-test-cluster` for end to end tests), which will use the Terraform configuration found in the `build/terraform/gke` directory.

You can customize the GKE cluster by appending the following parameters to your make target, via environment
variables, or by setting them within your
[`local-includes`](../building-testing.md#set-local-make-targets-and-variables-with-local-includes) directory. For end to end tests, update `GCP_CLUSTER_NAME` to e2e-test-cluster
in the Makefile.

See the table below for available customizations :

| Parameter                                      | Description                                                                           | Default         |
|------------------------------------------------|---------------------------------------------------------------------------------------|-----------------|
| `GCP_CLUSTER_NAME`                             | The name of the cluster                                                               | `test-cluster`  |
| `GCP_CLUSTER_ZONE` or `GCP_CLUSTER_LOCATION`   | The name of the Google Compute Engine zone/location in which the cluster will resides | `us-west1-c`    |
| `GCP_CLUSTER_NODEPOOL_AUTOSCALE`               | Whether or not to enable autoscaling on game server nodepool                          | `false`         |
| `GCP_CLUSTER_NODEPOOL_MIN_NODECOUNT`           | The number of minimum nodes if autoscale is enabled                                   | `1`             |
| `GCP_CLUSTER_NODEPOOL_MAX_NODECOUNT`           | The number of maximum nodes if autoscale is enabled                                   | `5`             |
| `GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT`        | The number of nodes to create in this cluster.                                        | `4`             |
| `GCP_CLUSTER_NODEPOOL_MACHINETYPE`             | The name of a Google Compute Engine machine type.                                     | `e2-standard-4` |
| `GCP_CLUSTER_NODEPOOL_ENABLEIMAGESTREAMING`    | Whether or not to enable image streaming for the `"default"` node pool in the cluster | `true`          |
| `GCP_CLUSTER_NODEPOOL_WINDOWSINITIALNODECOUNT` | The number of Windows nodes to create in this cluster.                                | `0`             |
| `GCP_CLUSTER_NODEPOOL_WINDOWSMACHINETYPE`      | The name of a Google Compute Engine machine type for Windows nodes.                   | `e2-standard-4` |

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
You can choose either Google Container Registry or Google Artifact Registry but you must set it explicitly.
For this guidance, we will use Google Artifact Registry.
You can [choose any registry region](https://cloud.google.com/artifact-registry/docs/docker/pushing-and-pulling)
but for this example, we'll just use `us-docker.pkg.dev`.
Please follow the [instructions](https://cloud.google.com/artifact-registry/docs/docker/pushing-and-pulling#before_you_begin) to create the registry
in your project properly before you contine.

In your shell, run `export REGISTRY=us-docker.pkg.dev/<YOUR-PROJECT-ID>/<YOUR-REGISTRY-NAME>` which will set the required `REGISTRY` parameters in our
Make targets. Then, to rebuild our images for this registry, we run `make build-images` again.

Before we can push the images, there is one more small step! So that we can run regular `docker push` commands
(rather than `gcloud docker -- push`), we have to authenticate against the registry, which will give us a short
lived key for our local docker config. To do this, run `make gcloud-auth-docker`, and now we have the short lived tokens.

To push our images up at this point, is simple `make push` and that will push up all images you just built to your
project's container registry.

Now that the images are pushed, to install the development version (with all imagePolicies set to always download),
run `make install` and Agones will install the image that you just built and pushed on the test cluster you
created at the beginning of this section. (if you want to see the resulting installation yaml, you can find it in `build/.install.yaml`)

Finally, to run all the end-to-end tests against your development version previously installed in your test cluster run
`make test-e2e` (this can also take a while). This will validate the whole application flow from start to finish. If
you're curious about how they work head to [tests/e2e](../test/e2e/).
Also [see the building guide](building-testing.md#running-individual-end-to-end-tests) for how to run individual end-to-end tests during development.

When you are finished, you can run `make clean-gcloud-test-cluster` (or
`make clean-gcloud-e2e-test-cluster`) to tear down your cluster.

## Running a Test Minikube cluster

This will setup a [Minikube](https://github.com/kubernetes/minikube) cluster, running on an `agones` profile,

Because Minikube runs on a virtualisation layer on the host (usually Docker), some of the standard build and development
 Make targets need to be replaced by Minikube specific targets.

First, [install Minikube](https://github.com/kubernetes/minikube#installation).

Next we will create the Agones Minikube cluster. Run `make minikube-test-cluster` to create the `agones` profile,
and a Kubernetes cluster of the supported version under this profile.

For e2e testing that requires multiple nodes, you can specify the number of nodes using the `MINIKUBE_NODES` environment variable.
A minimum of 2-3 nodes is recommended for comprehensive e2e testing:

```bash
make minikube-test-cluster MINIKUBE_NODES=3 
```

This will also install the kubectl authentication credentials in `~/.kube`, and set the
[`kubectl` context](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)
to `agones`.

Great! Now we are setup, let's try out the development shell, and see if our `kubectl` is working!

Run `make minikube-shell` to enter the development shell. You should see a bash shell that has you as the root user.
Enter `kubectl get pods` and press enter. You should see that you have no resources currently, but otherwise see no errors.
Assuming that all works, let's exit the shell by typing `exit` and hitting enter, and look at a building, pushing and
installing Agones on Minikube next.

You may remember in the first part of this walkthrough, we ran `make build-images`, which created all the images and binaries
we needed to work with Agones locally. We can push these images them straight into Minikube very easily!

Run `make minikube-push` which will send all of Agones's docker images from your local Docker into the Agones Minikube
instance.

Now that the images are pushed, to install the development version,
run `make minikube-install` and Agones will install the images that you built and pushed to the Agones Minikube instance
(if you want to see the resulting installation yaml, you can find it in `build/.install.yaml`).

It's worth noting that Minikube does let you [reuse its Docker daemon](https://minikube.sigs.k8s.io/docs/handbook/pushing/#1-pushing-directly-to-the-in-cluster-docker-daemon-docker-env),
and build directly on Minikube, but in this case this approach is far simpler,
and makes cross-platform support for the build system much easier.

To push your own images into the cluster, take a look at Minikube's
[Pushing Images](https://minikube.sigs.k8s.io/docs/handbook/pushing/) guide.

Running end-to-end tests on Minikube can be done via the `make minikube-test-e2e` target, but this can often overwhelm a local minikube cluster, so use at your own risk. Take a look at the [Building Guide](building-testing.md#running-individual-end-to-end-tests) to run individual tests on a case by case basis.

If you are getting issues connecting to `GameServers` running on minikube, check the
[Agones minikube](https://agones.dev/site/docs/installation/creating-cluster/minikube/) documentation. You may need to
change the driver version through the `MINIKUBE_DRIVER` variable. See the
[local-includes](building-testing.md#set-local-make-targets-and-variables-with-local-includes) on how to change this permanently on
your development machine.

## Running a Test Kind cluster

 This will setup a [Kubernetes IN Docker](https://github.com/kubernetes-sigs/kind) cluster named agones by default.

Because Kind runs on a docker on the host, some of the standard build and development Make targets
need to be replaced by kind specific targets.

First, [install Kind](https://github.com/kubernetes-sigs/kind#installation-and-usage).

Next we will create the Agones Kind cluster. Run `make kind-test-cluster` to create the `agones` Kubernetes cluster.

This will also setup helm and a kubeconfig, the kubeconfig location can found using the following command `kind get kubeconfig --name=agones` assuming you're using the default `KIND_PROFILE`.

You can verify that your new cluster information by running (if you don't have kubectl you can skip to the shell section):

```
KUBECONFIG=$(kind get kubeconfig-path --name=agones) kubectl cluster-info
```

Great! Now we are setup, we also provide a development shell with a handful set of tools like kubectl and helm.

Run `make kind-shell` to enter the development shell. You should see a bash shell that has you as the root user.
Enter `kubectl get pods` and press enter. You should see that you have no resources currently, but otherwise see no errors.
Assuming that all works, let's exit the shell by typing `exit` and hitting enter, and look at a building, pushing and installing Agones on Kind next.

You may remember in the first part of this walkthrough, we ran `make build-images`, which created all the images and binaries
we needed to work with Agones locally. We can push these images them straight into kind very easily!

Run `make kind-push` which will send all of Agones's docker images from your local Docker into the Agones Kind container.

Now that the images are pushed, to install the development version,
run `make kind-install` and Agones will install the images that you built and pushed to the Agones Kind cluster.

Running end-to-end tests on Kind is done via the `make kind-test-e2e` target. This target use the same `make test-e2e` but also setup some prerequisites for use with a Kind cluster.

**Note:** By default, KIND creates a single control-plane node, which is subject to the Kubernetes pod-per-node limit (about 110 pods). For e2e tests, you may want to create additional worker nodes to ensure enough pod availability. You can do this by providing a custom KIND config file when creating the cluster:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
   - role: control-plane
   - role: worker
   - role: worker
```

Create the cluster with:
```sh
kind create cluster --name agones --config <your-config.yaml>
```

This will ensure enough pod capacity for e2e tests.

If you are having performance issues, check out these docs [here](https://kind.sigs.k8s.io/docs/user/quick-start/#creating-a-cluster)

## Running a Custom Test Environment

This section is addressed to developers using a custom Kubernetes provider, a custom image repository and/or multiple test clusters.

Prerequisites:
- a(some) running k8s cluster(s)
- Have kubeconfig file(s) ready
- docker must be logged into the image repository you're going to use

To begin, you need to set up the following environment variables:
- `KUBECONFIG` should point to the kubeconfig file used to access the cluster;
   if unset, it defaults to `~/.kube/config`
- `REGISTRY` should point to your image repository of your choice (i.e. us-docker.pkg.dev/<YOUR-PROJECT-ID>/<YOUR-REGISTRY-NAME>)
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

If you need to clean up your cluster, you can use `make uninstall` to remove Agones.

## Next Steps

- See [Development Workflow Guide](development-workflow.md) for remote debugging and advanced development patterns
- See [Make Reference](make-reference.md) for detailed documentation of all cluster-specific make targets