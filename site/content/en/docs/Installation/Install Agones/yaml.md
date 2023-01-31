---
title: "Install Agones using YAML"
linkTitle: "YAML"
weight: 10
description: >
  We can install Agones to the cluster using an install.yaml file.
---

### Installing Agones

{{< alert title="Warning" color="warning">}}
Installing Agones with the `install.yaml` will set up the TLS certificates
stored in this repository for securing Kubernetes webhooks communication.

If you want to generate new certificates or use your own for production workloads,
we recommend using the [helm installation]({{< relref "helm.md" >}}).
{{< /alert >}}

```bash
kubectl create namespace agones-system
kubectl apply --server-side -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/install/yaml/install.yaml
```

To change the [configurable parameters](https://agones.dev/site/docs/installation/install-agones/helm/#configuration) in the `install.yaml` file, you can use helm directly to generate a custom file locally.

The following example sets the `featureGates` and `generateTLS` helm parameters in `install.yaml`:

```bash
helm pull --untar https://agones.dev/chart/stable/agones-{{< release-version >}}.tgz && \
cd agones && \
helm template agones-manual --namespace agones-system  . \
  --set agones.controller.generateTLS=false \
  --set agones.allocator.generateTLS=false \
  --set agones.allocator.generateClientTLS=false \
  --set agones.crds.cleanupOnDelete=false \
  --set agones.featureGates="PlayerTracking=true" \
  > install-custom.yaml
```
Note: `pull` command was introduced in Helm version 3.

You can also find the install.yaml in the latest `agones-install` zip from the [releases](https://github.com/googleforgames/agones/releases) archive.

### Uninstalling Agones

To uninstall/delete the `Agones` deployment and delete `agones-system` namespace:

```bash
kubectl delete fleets --all --all-namespaces
kubectl delete gameservers --all --all-namespaces
kubectl delete -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/install/yaml/install.yaml
kubectl delete namespace agones-system
```

Note: you should wait up to a couple of minutes until all resources described in `install.yaml` file would be deleted.

## Next Steps

- [Confirm Agones is up and running]({{< relref "../confirm.md" >}})
