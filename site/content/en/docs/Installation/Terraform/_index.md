---
title: "Deploy Kubernetes cluster and install Agones using Terraform"
linkTitle: "Install with Terraform"
weight: 50
description: >
  Install a [Kubernetes](http://kubernetes.io) cluster and Agones declaratively using Terraform.
---

## Prerequisites

{{% feature expiryVersion="1.7.0" %}}
- [Terraform](https://www.terraform.io/) v0.12.21
- [Helm](https://docs.helm.sh/helm/) package manager 2.10.0+
- Access to the the Kubernetes hosting provider you are using (e.g. `gcloud`,
  `awscli`, or `az` utility installed)
- Git
{{% /feature %}}

{{% feature publishVersion="1.7.0" %}}
- [Terraform](https://www.terraform.io/) v0.12.21
- Access to the the Kubernetes hosting provider you are using (e.g. `gcloud`,
  `awscli`, or `az` utility installed)
- Git

{{% alert color="info" title="Note" %}}
We recently updated all our Terraform modules and examples to use
a {{% ghlink href="install/terraform/modules/helm3" link_test="false" %}}Helm 3 Module{{% /ghlink %}}.

If you still require the {{% ghlink href="install/terraform/modules/helm" %}}Helm 2{{% /ghlink %}} module, it is still
available, but isn't being actively maintained.
{{% /alert %}}

{{% /feature %}}

