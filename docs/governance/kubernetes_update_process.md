
# Kubernetes versions update policy

We will support 3 releases of Kubernetes, targeting the newest version as being the [default version in the GKE Rapid channel](https://cloud.google.com/kubernetes-engine/docs/release-notes#current_versions) (since tests run on GKE we can't support versions that GKE doesn't support). However, we will ensure that at least one of the 3 versions chosen for each Agones release is supported by each of the major cloud providers (EKS and AKS). The vendored version of client-go will be aligned with the middle of the three supported Kubernetes versions.

# Kubernetes versions update process

1. Create a Issue from the [kubernetes update issue template](../../.github/ISSUE_TEMPLATE/kubernetes_update.md) with the newly supported versions.
2. Complete all items in the issue checklist.
3. Close the issue.
