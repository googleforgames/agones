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

You can also find the install.yaml in the latest `agones-install` zip from the [releases](https://github.com/googleforgames/agones/releases) archive.

## Next Steps

- [Confirm Agones is up and running]({{< relref "../confirm.md" >}})
