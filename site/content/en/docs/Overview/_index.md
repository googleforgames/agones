---
title: "Overview"
linkTitle: "Overview"
weight: 1
description: >
  Agones is a library for hosting, running and scaling [dedicated game servers](https://en.wikipedia.org/wiki/Game_server#Dedicated_server) on [Kubernetes](https://kubernetes.io).
---

_Agones, is derived from the Greek word agōn which roughly translates to “contest”, “competition at games” and “gathering”
([source](https://www.merriam-webster.com/dictionary/agones))._

## What is Agones?

Agones is an open source platform, for deploying, hosting, scaling, and orchestrating dedicated game servers for
large scale multiplayer games, built on top of the industry standard, distributed system platform [Kubernetes](https://kubernetes.io).

Agones replaces bespoke or proprietary cluster management and game server scaling solutions with an open source solution that
can be utilised and communally developed - so that you can focus on the important aspects of building a multiplayer game,
rather than developing the infrastructure to support it. 

Built with both Cloud and on-premises infrastructure in mind, Agones can adjust its strategies as needed
for Fleet management, autoscaling, and more to ensure the resources being used to host dedicated game servers are
cost optimal for the environment that they are in. 

## Why Agones?

Some of Agones' advantages:

- Lower development and operational costs for hosting, scaling and orchestrating multiplayer game servers.  
- Any game server that can run on Linux can be hosted and orchestrated on Agones - in any language, or set of dependencies.
- Run Agones anywhere Kubernetes can run - in the cloud, on premise, on your local machine or anywhere else you need it.
- Game services and your game servers can be on the same foundational platform - simplifying your tooling and your operations knowledge.
- By extending Kubernetes, Agones allows you to take advantage of the thousands of developers that have worked on the features of Kubernetes, and the ecosystem of tools that surround it.
- Agones is free, open source and developed entirely in the public. Help shape its future by getting involved with the community.

## Major Features

Agones incorporates these abilities:

- Agones extends Kubernetes, such that it gets native abilities to create, run, manage and scale dedicated game server processes within
  Kubernetes clusters using standard Kubernetes tooling and APIs.
- Run and update Fleets of Game Servers, without worrying about having Game Servers shutdown that have active players
 on them.
- Deploy game servers inside a [Docker container](https://www.docker.com/resources/what-container), with any combination of dependencies or binaries.
- Integrated game server SDK for game server lifecycle managements, including health checking, state management, configuration and more.
- Autoscaling capabilities to ensure players always have a game server available to play on.
- Out of the box metrics and log aggregation to track and visualise what is happening across all your game server sessions.
- Modular architecture that can be tailored to your specific multiplayer game mechanics.
- Game server scheduling and allocation strategies to ensure cost optimisation across cloud and on-premise environments. 

## Code of Conduct

Participation in this project comes under the {{< ghlink href="code-of-conduct.md" branch="main" >}}Contributor Covenant Code of Conduct{{< /ghlink >}}

## What Next?
- Review our [Prerequisite Knowledge]({{% ref "/docs/Prerequisite Knowledge/_index.md" %}}). Especially if the above
  sounds fantastic, but you aren't yet familiar with technology like Kubernetes or terms such as "Game Servers".
- Have a look at our [installation guides]({{< ref "/docs/Installation/_index.md" >}}), for setting up a Kubernetes cluster
  and installing Agones on it.
- Go through our [Quickstart Guides]({{< ref "/docs/Getting Started/_index.md" >}}) to take you through setting up a simple game server on Agones.
