---
title: "Overview"
linkTitle: "Overview"
weight: 1
description: >
  Agones is a library for hosting, running and scaling [dedicated game servers](https://en.wikipedia.org/wiki/Game_server#Dedicated_server) on [Kubernetes](https://kubernetes.io).
---

_Agones, is derived from the Greek word agōn which roughly translates to “contest”, “competition at games” and “gathering”
([source](https://www.merriam-webster.com/dictionary/agones))._

## Disclaimer
This software is currently alpha, and subject to change. Not to be used in production systems.

## Why does this project exist?
Agones replaces usual bespoke or proprietary cluster management and game server scaling solutions with a [Kubernetes](https://kubernetes.io/) cluster
that includes the Agones custom _[Kubernetes Controller](https://kubernetes.io/docs/concepts/api-extension/custom-resources/#custom-controllers)_ and matching [Custom Resource Definitions](https://kubernetes.io/docs/concepts/api-extension/custom-resources/#customresourcedefinitions) for _GameServers_, _Fleets_ and more.

With Agones, Kubernetes gets native abilities to create, run, manage and scale dedicated game server processes within Kubernetes clusters using standard Kubernetes tooling and APIs. This model also allows any matchmaker to interact directly with Agones via the Kubernetes API to provision a dedicated a game server.

For more details on why this project was written, read the
[announcement blog post](https://cloudplatform.googleblog.com/2018/03/introducing-Agones-open-source-multiplayer-dedicated-game-server-hosting-built-on-Kubernetes.html).

## Major Features
- Define a single `GameServer`, and/or large game server `Fleets` within Kubernetes - either through yaml or via the API
- Manage GameServer lifecycles - including health checking and connection information.
- `Fleet` Autoscaling capabilities that integrate with Kubernetes' native cluster autoscaling
- Gameserver specific metric exports and dashboards for ops teams

## Requirements
- Kubernetes cluster version 1.11+
    - [Minikube](https://github.com/kubernetes/minikube), [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/) and [Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/) have been tested
    - If you are creating and managing your own Kubernetes cluster, the
    [MutatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#mutatingadmissionwebhook-beta-in-19), and
    [ValidatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#validatingadmissionwebhook-alpha-in-18-beta-in-19)
    admission controllers are required.
    We also recommend following the
    [recommended set of admission controllers](https://kubernetes.io/docs/admin/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use).
- Firewall access for the range of ports that Game Servers can be connected to in the cluster.
- Game Servers must have the [project SDK]({{< ref "/docs/Guides/Client SDKs/_index.md"  >}}) integrated, to manage Game Server state, health checking, etc.

## Code of Conduct

Participation in this project comes under the {{< ghlink href="code-of-conduct.md" branch="master" >}}Contributor Covenant Code of Conduct{{< /ghlink >}}

## This all sounds great, but can you explain Docker and/or Kubernetes to me?

### Docker
- [Docker's official "Getting Started" guide](https://docs.docker.com/get-started/)
- [Katacoda's free, interactive Docker course](https://www.katacoda.com/courses/docker)

### Kubernetes
- [You should totally read this comic, and interactive tutorial](https://cloud.google.com/kubernetes-engine/kubernetes-comic/)
- [Katacoda's free, interactive Kubernetes course](https://www.katacoda.com/courses/kubernetes)