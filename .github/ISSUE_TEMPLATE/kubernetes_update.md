---
name: Upgrade Kubernetes Version
about: Issue for updating the Kubernetes version (usually decided in community meetings).
title: 'Update Kubernetes to {version}'
labels: kind/operations, kind/breaking
assignees: ''

---

List of items to do for upgrading to {version}:

- [ ] Update terraform submodules
    - [ ] GKE
    - [ ] Azure
    - [ ] EKS
- [ ] Update prow cluster (even though we aren't using it yet, we should keep it in sync)
    - [ ] Recreate cluster with new scripts: `cd build/terraform/prow; terraform apply -var project=agones-images`
- [ ] Update e2e cluster
    - [ ] Recreate cluster with new scripts: `cd build/terraform/e2e; terraform apply -var project=agones-images`
- [ ] Update kubectl in dev tooling
    - [ ] Update kubectl in `build/build-image/Dockerfile`
    - [ ] Update kubectl in `build/e2e-image/Dockerfile`
- [ ] Update documentation for creating clusters
    - [ ] Config.toml `supported_k8s` and related (do `dev_` before main)
- [ ] Update the dev tooling to create clusters
    - [ ] Minikube
    - [ ] Kind
- [ ] Update the k8s image used in the helm [pre-delete-hook](https://github.com/googleforgames/agones/blob/main/install/helm/agones/templates/hooks/pre_delete_hook.yaml)
- [ ] Update client-go
- [ ] Update CRD API reference
    - [ ] Update links to k8s documentation in `site/assets/templates/crd-doc-config.json`
    - [ ] Regenerate crd api reference docs - `make gen-api-docs`
- [ ] Regenerate Kubernetes resource includes (e.g. ObjectMeta, PodTemplateSpec)
    - [ ] Start a cluster with `make gcloud-test-cluster`, uninstall agones using `helm uninstall agones -n agones-system`, and then run  `make gen-embedded-openapi` and `make gen-install`
- [ ] If client-go pulled in a new version of gRPC, then also
    - [ ] Update the SDK [base image grpc version](https://github.com/googleforgames/agones/blob/main/build/includes/sdk.mk#L30) and rebuild the image. Note that this can take a while and in the past we have had to manually push it to gcr because cloud build doesn't like how long it takes.
    - [ ] Regenerate allocated API endpoints: [make gen-allocation-grpc](https://github.com/googleforgames/agones/blob/main/build/includes/allocation.mk#L55)
    - [ ] Regenerate all client sdks: [make gen-all-sdk-grpc](https://github.com/googleforgames/agones/blob/main/build/README.md#make-gen-all-sdk-grpc)
    - [ ] Update the version number in C++ Cmake scripts [here](https://github.com/googleforgames/agones/blob/main/sdks/cpp/CMakeLists.txt#L100) and [here](https://github.com/googleforgames/agones/blob/main/sdks/cpp/cmake/prerequisites.cmake#L34)
