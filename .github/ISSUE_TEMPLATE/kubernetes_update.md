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
    - [ ] Update Kubernetes version of GKE cluster to {version_2}
    - [ ] Update Kubernetes version of AKS to the newest supported version in {version_1} {version_2} {version_3}
    - [ ] Update Kubernetes version of EKS to the newest supported version in {version_1} {version_2} {version_3}
- [ ] Update kubectl in dev tooling to {version_2}
    - [ ] Update kubectl in `build/build-image/Dockerfile`
    - [ ] Update kubectl in `build/e2e-image/Dockerfile`
- [ ] Update the Kubernetes version of the below test clusters to {version_2}
    - [ ] Minikube (Get the patch version [here](https://github.com/kubernetes/minikube/blob/master/CHANGELOG.md))
    - [ ] Kind (Confirm {version_2} is supported and get the patch version [here](https://github.com/kubernetes-sigs/kind/releases))
- [ ] Update the k8s image used in the helm [pre-delete-hook](https://github.com/googleforgames/agones/blob/main/install/helm/agones/templates/hooks/pre_delete_hook.yaml) to {version_2}
- [ ] Update client-go in `go.mod` and `test/terraform/go.mod` to {version_2} and re-run `go mod tidy` and `go mod vendor`
- [ ] Update CRD API reference to {version_2}
    - [ ] Update links to k8s documentation in `site/assets/templates/crd-doc-config.json`
    - [ ] Regenerate crd api reference docs - `make gen-api-docs`
    - [ ] Regenerate crd client libraries - `make gen-crd-client`
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
    - [ ] Update the SDK [base image grpc version](https://github.com/googleforgames/agones/blob/main/build/includes/sdk.mk#L30) and rebuild the image. Note that this can take a while and in the past we have had to manually push it to gcr because cloud build doesn't like how long it takes.
    - [ ] Regenerate allocated API endpoints: [make gen-allocation-grpc](https://github.com/googleforgames/agones/blob/main/build/includes/allocation.mk#L55)
    - [ ] Regenerate all client sdks: [make gen-all-sdk-grpc](https://github.com/googleforgames/agones/blob/main/build/README.md#make-gen-all-sdk-grpc)
    - [ ] Update the version number in C++ Cmake scripts [here](https://github.com/googleforgames/agones/blob/main/sdks/cpp/CMakeLists.txt#L100) and [here](https://github.com/googleforgames/agones/blob/main/sdks/cpp/cmake/prerequisites.cmake#L34)
- [ ] Confirm the update works as expected by running e2e tests
    - [ ] Update the Kubernetes version of the e2e clusters
        - [ ] In `terraform/e2e/module.tf`, update variable `kubernetes_versions_standard` and `kubernetes_versions_autopilot` to the new versions to be supported
        - [ ] Recreate cluster with new scripts: `cd build/terraform/e2e; terraform init -backend-config="bucket=agones-images-e2e-infra-bucket-tfstate" -backend-config="prefix=terraform/state"; terraform apply -var project=agones-images`
    - [ ] Update the Cloud Build configuration to run e2e test on the new created clusters
        - [ ] Update the `versionsAndRegions` variable to reflect new versions in `cloudbuild.yaml` `submit-e2e-test-cloud-build` step
        - [ ] Submit a PR to trigger the e2e tests and verfiy they all pass
