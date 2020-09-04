---
title: "Install and configure Agones on Kubernetes"
linkTitle: "Installation"
weight: 4
description: >
  Instructions for creating a Kubernetes cluster and installing Agones.
---

## Usage Requirements

{{% feature expiryVersion="1.9.0" %}}
- **Kubernetes cluster version 1.15**
{{% /feature %}}
{{% feature publishVersion="1.9.0" %}}
- **Kubernetes cluster version 1.16**
{{% /feature %}}
    - [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/),
      [Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/),
      [Amazon EKS](https://aws.amazon.com/eks/) and [Minikube](https://github.com/kubernetes/minikube) have been tested
    - If you are creating and managing your own Kubernetes cluster, the
    [MutatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#mutatingadmissionwebhook-beta-in-19), and
    [ValidatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#validatingadmissionwebhook-alpha-in-18-beta-in-19)
    admission controllers are required.
       - We also recommend following the
    [recommended set of admission controllers](https://kubernetes.io/docs/admin/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use).
- Firewall access for the range of ports that Game Servers can be connected to in the cluster.
- Game Servers must have the [game server SDK]({{< ref "/docs/Guides/Client SDKs/_index.md"  >}}) integrated, to manage Game Server state, health checking, etc.

{{% feature expiryVersion="1.9.0" %}}
{{< alert title="Warning" color="warning">}}
Later versions of Kubernetes may work, but this project is tested against 1.15, and is therefore the supported version.
Agones will update its support to n-1 version of what is available across the majority of major cloud providers - GKE, EKS and
AKS, while also ensuring that all Cloud providers can support that version.
{{< /alert >}}
{{% /feature %}}
{{% feature publishVersion="1.9.0" %}}
{{< alert title="Warning" color="warning">}}
Later versions of Kubernetes may work, but this project is tested against 1.16, and is therefore the supported version.
Agones will update its support to n-1 version of what is available across the majority of major cloud providers - GKE, EKS and
AKS, while also ensuring that all Cloud providers can support that version.
{{< /alert >}}
{{% /feature %}}

{{< alert title="Note" color="info">}}
When running in production, Agones should be scheduled on a dedicated pool of nodes, distinct from where Game Servers
are scheduled for better isolation and resiliency. By default Agones prefers to be scheduled on nodes labeled with
`agones.dev/agones-system=true` and tolerates the node taint `agones.dev/agones-system=true:NoExecute`.
If no dedicated nodes are available, Agones will run on regular nodes.
{{< /alert >}}
