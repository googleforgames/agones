---
title: "Install Agones using YAML"
linkTitle: "YAML"
weight: 10
description: >
  We can install Agones to the cluster using an install.yaml file.
---

{{< alert title="Warning" color="warning">}}
Installing Agones with the `install.yaml` will setup the TLS certificates stored in this repository for securing
kubernetes webhooks communication. 

If you want to generate new certificates or use your own for production workloads,
we recommend using the [helm installation]({{< relref "helm.md" >}}).
{{< /alert >}}

```bash
kubectl create namespace agones-system
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/install/yaml/install.yaml
```

If you want to change the parameters in the `install.yaml` file, you can use helm directly to generate a custom file locally, but make sure new parameters correspond to the [following ones](https://agones.dev/site/docs/installation/install-agones/helm/#configuration).

Example of setting `featureGates` and `generateTLS` helm parameters in `install.yaml`:
```
helm pull --untar https://agones.dev/chart/stable/agones-{{< release-version >}}.tgz && \
cd agones && \
helm template agones-manual --namespace agones-system  . \
  --set agones.controller.generateTLS=false \
  --set agones.allocator.generateTLS=false \
  --set agones.crds.cleanupOnDelete=false \
  --set agones.featureGates="PlayerTracking=true" \
  > install-custom.yaml
```
Note: `pull` command was introduced in Helm version 3.

You can also find the install.yaml in the latest `agones-install` zip from the [releases](https://github.com/googleforgames/agones/releases) archive.

## Next Steps

- [Confirm Agones is up and running]({{< relref "../confirm.md" >}})
