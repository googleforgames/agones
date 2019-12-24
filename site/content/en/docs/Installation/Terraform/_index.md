---
title: "Deploy Kubernetes cluster and install Agones using Terraform"
linkTitle: "Install with Terraform"
weight: 50
description: >
  Install a [Kubernetes](http://kubernetes.io) cluster and Agones declaratively using Terraform.
---

## Prerequisites

- [Terraform](https://www.terraform.io/) v0.12.3
- [Helm](https://docs.helm.sh/helm/) package manager 2.10.0+
- Access to the the Kubernetes hosting provider you are using (e.g. `gcloud`
{{% feature publishVersion="1.3.0" %}}, `awscli`{{% /feature %}} or `az` utility installed)
- Git
