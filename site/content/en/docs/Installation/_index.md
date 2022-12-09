---
title: "Install and configure Agones on Kubernetes"
linkTitle: "Installation"
weight: 4
description: >
  Instructions for creating a Kubernetes cluster and installing Agones.
---

## Usage Requirements

- **Kubernetes cluster version {{% k8s-version %}}**
  - [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/),
    [Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/),
    [Amazon EKS](https://aws.amazon.com/eks/) and [Minikube](https://github.com/kubernetes/minikube) are supported.
  - If you are creating and managing your own Kubernetes cluster, the
    [MutatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#mutatingadmissionwebhook-beta-in-19), and
    [ValidatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#validatingadmissionwebhook-alpha-in-18-beta-in-19)
    admission controllers are required.
    - We also recommend following the
      [recommended set of admission controllers](https://kubernetes.io/docs/admin/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use).
- Firewall access for the range of ports that Game Servers can be connected to in the cluster.
- Game Servers must have the [game server SDK]({{< ref "/docs/Guides/Client SDKs/_index.md"  >}}) integrated, to manage Game Server state, health checking, etc.

{{< alert title="Warning" color="warning">}}
Later versions of Kubernetes may work, but this project is tested against {{% k8s-version %}}, and is therefore the supported version.
Agones will update its support to the n-1 version of what is available across the majority of major cloud providers - GKE, EKS and
AKS, while also ensuring that all Cloud providers can support that version.
{{< /alert >}}

## Supported Container Architectures

The following container operating systems and architectures can be utilised with Agones:

| OS        | Architecture | Support    |
| --------- | ------------ | ---------- |
| linux     | `amd64`      | **Stable** |
| linux     | `arm64`      | Alpha      |
| [windows] | `amd64`      | Alpha      |

For all the platforms in Alpha, we would appreciate testing and bug reports on any issue found.

## Agones and Kubernetes Supported Versions

Each version of Agones supports a specific version of Kubernetes. When a new version of Agones supports a new version of Kubernetes, it is explicitly called out in the [release notes](https://agones.dev/site/blog/releases/).

The following table lists recent Agones versions and their corresponding required Kubernetes versions:

| Agones version | Kubernetes version |
| -------------- | ------------------ |
| 1.28           | 1.23               |
| 1.27           | 1.23               |
| 1.26           | 1.23               |
| 1.25           | 1.22               |
| 1.24           | 1.22               |
| 1.23           | 1.22               |
| 1.22           | 1.21               |
| 1.21           | 1.21               |

## Best Practices

### Separation of Agones from GameServer nodes

When running in production, Agones should be scheduled on a dedicated pool of nodes, distinct from where Game Servers
are scheduled for better isolation and resiliency. By default Agones prefers to be scheduled on nodes labeled with
`agones.dev/agones-system=true` and tolerates the node taint `agones.dev/agones-system=true:NoExecute`.
If no dedicated nodes are available, Agones will run on regular nodes.

[windows]: {{% ref "/docs/Guides/windows-gameservers.md" %}}
