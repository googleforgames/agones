---
name: Upgrade Supported Kubernetes Versions
about: Issue for updating the supported Kubernetes versions.
title: 'Update Supported Kubernetes to {version_1} {version_2} {version_3}'
labels: kind/operations, kind/breaking
assignees: ''

---

Agones supports and is tested against 3 releases of Kubernetes, targeting the newest version as being the [default version in the GKE Rapid channel](https://cloud.google.com/kubernetes-engine/docs/release-notes#current_versions). The vendored version of client-go will be aligned with the middle of the three supported Kubernetes versions ({version_2}). All the example clusters will use the middle of the three supported Kubernetes versions ({version_2}).

List of items to do for upgrading to {version_1} {version_2} {version_3}

- [ ] Update the cluster version of terraform submodules in `install/terraform/modules`
    - [ ] Update Kubernetes version of GKE cluster (both `gke` and `gke-autopilot`) to {version_2}
    - [ ] Update Kubernetes version of AKS to the newest supported version in {version_1} {version_2} {version_3}
    - [ ] Update Kubernetes version of EKS to the newest supported version in {version_1} {version_2} {version_3}
- [ ] Update kubectl in dev tooling to {version_2}, the latest patch version can be found [here](https://kubernetes.io/releases/)
    - [ ] Update kubectl in `build/build-image/Dockerfile`
    - [ ] Update kubectl in `build/e2e-image/Dockerfile`
- [ ] Update the Kubernetes version of the below test clusters to {version_2}
    - [ ] Minikube in `build/includes/minikube.mk` (Get the patch version [here](https://kubernetes.io/releases/) since minikube supports the latest Kubernetes release)
    - [ ] Kind in `build/includes/kind.mk` (Confirm {version_2} is supported and get the patch version [here](https://github.com/kubernetes-sigs/kind/releases))
- [ ] Update the k8s image used in the helm [pre-delete-hook](https://github.com/googleforgames/agones/blob/main/install/helm/agones/templates/hooks/pre_delete_hook.yaml) to {version_2} (Get the patch version [here](https://hub.docker.com/r/bitnami/kubectl))
- [ ] Update client-go in `go.mod` to {version_2} by running `go get k8s.io/client-go@{CORRESPONDING_VERSION}` and `go get k8s.io/apiextensions-apiserver@{CORRESPONDING_VERSION}`, then re-run `go mod tidy` and `go mod vendor`
- [ ] Update CRD API reference to {version_2}
    - [ ] Update links to k8s documentation in `site/assets/templates/crd-doc-config.json`
    - [ ] Regenerate crd api reference docs - `make gen-api-docs`
    - [ ] Regenerate crd client libraries and generated code - `make gen-crd-code`
- [ ] Regenerate Kubernetes resource includes (e.g. ObjectMeta, PodTemplateSpec)
    - [ ] Start a cluster with `make gcloud-test-cluster` (this cluster will use Kubernetes {version_2}), uninstall agones using `helm uninstall agones -n agones-system`, and then run  `make gen-embedded-openapi` and `make gen-install`
- [ ] Update documentation for creating clusters and k8s API references to align with the above clusters versions and the k8s API version
    - [ ] `site/config.toml`
        - [ ] `dev_supported_k8s`, which are {version_1} {version_2} {version_3}
        - [ ] `dev_k8s_api_version`, which is {version_2}
        - [ ] `dev_gke_example_cluster_version`, which is {version_2}
        - [ ] `dev_aks_example_cluster_version`, which is the newest AKS supported version in {version_1} {version_2} {version_3}
        - [ ] `dev_eks_example_cluster_version`, which is the newest EKS supported version in {version_1} {version_2} {version_3}
        - [ ] `dev_minikube_example_cluster_version`, which is {version_2} with the supported patch version
- [ ] If client-go pulled in a new version of gRPC, then also
    - [ ] Update the `grpc_release_tag` in the SDK [base image grpc version](https://github.com/googleforgames/agones/blob/main/build/includes/sdk.mk).
    - [ ] Update the gRPC version number in C++ gRPC Dependency documentation [here](https://github.com/googleforgames/agones/blob/main/site/content/en/docs/Guides/Client%20SDKs/cpp.md).
    - [ ] Update the gRPC version
      ([Dockerfile](https://github.com/googleforgames/agones/blob/main/examples/cpp-simple/Dockerfile)) and
      increment the image tag
      ([Makefile](https://github.com/googleforgames/agones/blob/main/examples/cpp-simple/Makefile)) in the C++
      `cpp-simple` example.
    - [ ] Regenerate all client sdks: [make gen-all-sdk-grpc](https://github.com/googleforgames/agones/blob/main/build/README.md#make-gen-all-sdk-grpc)
          This can take 20 minutes or so, as the above changes force a rebuild. Plan your day accordingly 😃.
    - [ ] Regenerate allocated API endpoints: [make gen-allocation-grpc](https://github.com/googleforgames/agones/blob/main/build/README.md#make-gen-allocation-grpc)
- [ ] Confirm the update works as expected by running e2e tests
    - [ ] Add the new supported Kubernetes versions to the e2e clusters creation
        - [ ] In `build/terraform/e2e/module.tf`, add the new supported version to the map `kubernetes_versions` following the instructions in the comment. We maintain sufficient quota for `CPUs` and `In-use Addresses` in `us-east1`, `us-west1`, `europe-west1`, and `asia-east1`, but we only have capacity for one version per region - follow the instructions to rotate the regions through new versions. (Keep the clusters on `RAPID` - we dogfood early versions in CI.)
        - [ ] Recreate clusters with new scripts: `cd build; make GCP_PROJECT=agones-images gcloud-e2e-test-cluster`
    - [ ] Update the Cloud Build configuration to run e2e test on the new created clusters, and disable the e2e test on the cluster with the oldest supported K8s version
        - [ ] Update the `versionsAndRegions` variable to add the new supported version and remove the oldest supported K8s version in `cloudbuild.yaml` `submit-e2e-test-cloud-build` step
        - [ ] Update the `featureWithGateByVersion` variable to add the new supported version and remove the oldest supported K8s version in `cloudbuild.yaml` `submit-e2e-test-cloud-build` step
        - [ ] Run `make lint` for code quality check.
        - [ ] Submit a PR to trigger the e2e tests and verfiy they all pass
    - [ ] After the PR that includes the above Cloud Build configuration change has been merged and all the existing pending PRs in the Cloud Build queue have picked up the new configuration, submit a separate PR to update the e2e clusters terraform module to remove the e2e cluster with the oldest supported K8s version.
        - [ ] In `build/terraform/e2e/module.tf`, continue following the instructions in the comment to update the `kubernetes_versions` map.
        - [ ] Destroy the old clusters with new scripts: `cd build; make GCP_PROJECT=agones-images gcloud-e2e-test-cluster`
- [ ] Recreate the performance test cluster, and config the performance test to run on the new cluster
    - [ ] In `build/terraform/performance/module.tf`, update the `kubernetes_versions` to {version_2} and make sure the region is always set to `us-central1`.
    - [ ] Recreate the cluster with the new script:
        ```
        cd build; make shell; cd build/terraform/performance
        terraform init -backend-config="bucket=agones-images-performance-infra-bucket-tfstate" -backend-config="prefix=terraform/state"
        terraform apply -var project="agones-images"
        ```
    - [ ] Update the `_TEST_CLUSTER_NAME` in `ci/perf-test-cloudbuild.yaml` to the name of the new created performance test cluster, and send a PR with the changes.
