---
title: "Deploy Kubernetes cluster and install Agones using Terraform"
linkTitle: "Install with Terraform"
weight: 50
description: >
  Install a [Kubernetes](http://kubernetes.io) cluster and Agones declaratively using Terraform.
---

## Prerequisites

{{% feature expiryVersion="1.19.0" %}}
- [Terraform](https://www.terraform.io/) v0.12.21
- Access to the the Kubernetes hosting provider you are using (e.g. `gcloud`,
  `awscli`, or `az` utility installed)
- Git
{{% /feature %}}
{{% feature publishVersion="1.19.0" %}}
- [Terraform](https://www.terraform.io/) v1.0.8
- Access to the the Kubernetes hosting provider you are using (e.g. `gcloud`,
  `awscli`, or `az` utility installed)
- Git
{{% /feature %}}

{{% alert color="info" title="Note" %}}
All our Terraform modules and examples use a {{% ghlink href="install/terraform/modules/helm3" %}}Helm 3 Module{{% /ghlink %}}.

The last Agones release to include a Helm 2 module was [1.9.0](https://agones.dev/site/blog/2020/09/29/1.9.0-kubernetes-1.16-nuget-and-tcp-udp/).
{{% /alert %}}
