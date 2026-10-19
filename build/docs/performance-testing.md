# Performance Testing Guide

This guide is a manual runbook for provisioning, running, and tearing down the Agones allocation
performance tests.

> [!WARNING]
> **The performance test cluster is large and expensive.** The default configuration in
> `build/terraform/performance/module.tf` creates a regional GKE cluster with 200 `e2-standard-4`
> nodes **per zone** (roughly 600 nodes in a three-zone region). There is no continuously running
> performance cluster and no automated Cloud Build trigger. Plan and budget each run, and tear the
> cluster down as soon as you are done.

The following are kept in the repository so the tests can be run when there is a planned need:

* `build/terraform/performance/` - Terraform for the cluster (`module.tf`) and its state bucket (`state-bucket/`)
* `ci/perf-test-cloudbuild.yaml` - Cloud Build config that builds, installs Agones and runs the test
* `build/performance-test.sh` - entrypoint script that creates the fleet and autoscaler and runs the scenario
* `test/load/allocation/` - the allocation load test itself

The examples below use the `agones-images` project. Replace it with your own project if you are
running the tests somewhere else.

## 1. Bootstrap the Terraform state bucket

The cluster's Terraform state is stored in a GCS bucket named `<PROJECT>-performance-infra-bucket-tfstate`.
Create it first:

```bash
cd build; make shell
cd build/terraform/performance/state-bucket
terraform init
terraform apply -var project="agones-images"
```

## 2. Create the performance test cluster

Check `kubernetes_versions` in `build/terraform/performance/module.tf` (the region should be
`us-central1`), then create the cluster:

```bash
cd build; make shell
cd build/terraform/performance
terraform init -backend-config="bucket=agones-images-performance-infra-bucket-tfstate" -backend-config="prefix=terraform/state"
terraform apply -var project="agones-images"
```

The cluster is named `standard-performance-test-cluster-<version>`, for example
`standard-performance-test-cluster-1-35`. It must match `_TEST_CLUSTER_NAME` in
`ci/perf-test-cloudbuild.yaml`.

> [!NOTE]
> `module.tf` also declares the `gke-game-server-firewall` and `gke-game-server-firewall-tcp`
> rules, which `build/terraform/e2e/` declares as well. In a project that also runs the e2e
> clusters (such as `agones-images`) these rules already exist. Import them into the performance
> state, or `-target` the cluster module, and see the teardown section before destroying.

```bash
# Only if the firewall rules already exist in the project:
terraform import -var project="agones-images" google_compute_firewall.udp projects/agones-images/global/firewalls/gke-game-server-firewall
terraform import -var project="agones-images" google_compute_firewall.tcp projects/agones-images/global/firewalls/gke-game-server-firewall-tcp
```

## 3. Run the performance test

### Option A: Cloud Build (recommended)

From the repository root, submit the Cloud Build config manually. It builds and pushes the images,
installs Agones on the performance cluster, and runs `build/performance-test.sh`:

```bash
gcloud builds submit . --project=agones-images --config=ci/perf-test-cloudbuild.yaml
```

Any of the `_TEST_*` substitutions in `ci/perf-test-cloudbuild.yaml` can be overridden, for example:

```bash
gcloud builds submit . --project=agones-images --config=ci/perf-test-cloudbuild.yaml \
  --substitutions=_TEST_DURATION=10m,_TEST_CLIENTS=50,_TEST_INTERVAL=1000
```

Follow the build in the Cloud Build console or with `gcloud builds log --stream <BUILD_ID>`.
The scenario results are printed at the end of the `performance-test` step.

### Option B: Run `build/performance-test.sh` against your own cluster

You can also run the test against any standard GKE cluster with Agones installed:

1. Create a standard GKE cluster: https://agones.dev/site/docs/installation/creating-cluster/gke/
2. Install Agones: https://agones.dev/site/docs/installation/install-agones/helm/
3. Optionally, add a node pool for Prometheus and install Prometheus and Grafana
   (https://agones.dev/site/docs/guides/metrics/#installation):

   ```bash
   gcloud container node-pools create agones-metrics --cluster={CLUSTER_NAME} --zone={REGION} \
     --node-taints agones.dev/agones-metrics=true:NoExecute \
     --node-labels agones.dev/agones-metrics=true \
     --num-nodes=1 \
     --machine-type=e2-standard-4
   ```

4. The script takes its settings as positional arguments, in this order:

   ```
   CLUSTER_NAME=agones-standard
   CLUSTER_LOCATION=us-central1-c
   REGISTRY=us-east1-docker.pkg.dev/user/agones
   PROJECT=my-project
   REPLICAS=10000
   AUTO_SHUTDOWN_DELAY=60
   BUFFER_SIZE=9900
   MAX_REPLICAS=20000
   DURATION=10m
   CLIENTS=50
   INTERVAL=1000
   ```

5. The script is written to run inside Cloud Build. To run it locally, comment out the lines that
   come after the variables are set, and change the `cd` directory:

   ```bash
   # export SHELL="/bin/bash"
   # mkdir -p /go/src/agones.dev/
   # ln -sf /workspace /go/src/agones.dev/agones
   # cd /go/src/agones.dev/agones/build

   # gcloud config set project $PROJECT
   # gcloud container clusters get-credentials $CLUSTER_NAME \
   #        --zone=$CLUSTER_LOCATION --project=$PROJECT

   # make install LOG_LEVEL=info REGISTRY='"'$REGISTRY'"' DOCKER_RUN=""

   # cd /go/src/agones.dev/agones/test/load/allocation
   cd ../test/load/allocation
   ```

6. From `agones/build`, run:

   ```bash
   sh performance-test.sh agones-standard us-central1-c us-east1-docker.pkg.dev/user/agones my-project \
     10000 60 9900 20000 10m 50 1000
   ```

   If you see timeout issues, re-run the command.

### Test configuration

The script is an entrypoint for the allocation performance test in `test/load/allocation`. The
fleet and autoscaler configuration (buffer size, min/max replicas, etc.) is in:

* [performance-test-fleet-template.yaml](https://github.com/agones-dev/agones/blob/main/test/load/allocation/performance-test-fleet-template.yaml)
* [performance-test-autoscaler-template.yaml](https://github.com/agones-dev/agones/blob/main/test/load/allocation/performance-test-autoscaler-template.yaml)

If `AUTO_SHUTDOWN_DELAY` is greater than zero, the simple game server shuts itself down that many
seconds after it is allocated.

`CLIENTS` and `INTERVAL` set the offered load. For example, 50 clients with a 500ms interval means
each client sends 2 allocation requests per second, so the allocator receives 50 * 2 = 100
requests per second.

## 4. Tear down the cluster and state bucket

Destroy the cluster as soon as the run is finished.

If the project also runs the e2e clusters, first remove the shared firewall rules from the
performance state so that `terraform destroy` does not delete rules the e2e clusters depend on:

```bash
cd build; make shell
cd build/terraform/performance
terraform init -backend-config="bucket=agones-images-performance-infra-bucket-tfstate" -backend-config="prefix=terraform/state"
terraform state list
terraform state rm google_compute_firewall.udp google_compute_firewall.tcp   # only if they are listed
terraform destroy -var project="agones-images"
```

Then delete the state bucket. It is versioned and `force_destroy` is `false`, so empty it
(including old object versions) before deleting it:

```bash
gcloud storage rm --recursive --all-versions gs://agones-images-performance-infra-bucket-tfstate/**
gcloud storage buckets delete gs://agones-images-performance-infra-bucket-tfstate
```

Confirm that nothing is left behind:

```bash
gcloud container clusters list --project=agones-images --filter="name~performance"
gcloud compute disks list --project=agones-images --filter="name~performance"
gcloud compute forwarding-rules list --project=agones-images
```

## Next Steps

- See [Cluster Setup Guide](cluster-setup.md) for setting up GKE clusters
- See [Make Reference](make-reference.md) for performance-related make targets
