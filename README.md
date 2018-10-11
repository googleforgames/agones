# Agones
[![GoDoc](https://godoc.org/agones.dev/agones?status.svg)](https://godoc.org/agones.dev/agones)
[![Go Report Card](https://goreportcard.com/badge/github.com/GoogleCloudPlatform/agones)](https://goreportcard.com/report/github.com/GoogleCloudPlatform/agones)
[![GitHub release](https://img.shields.io/github/release/GoogleCloudPlatform/agones.svg)](https://github.com/GoogleCloudPlatform/agones/releases)
[![Follow on Twitter](https://img.shields.io/twitter/follow/agonesdev.svg?style=social&logo=twitter)](https://twitter.com/intent/follow?screen_name=agonesdev)

Agones is a library for hosting, running and scaling [dedicated game servers](https://en.wikipedia.org/wiki/Game_server#Dedicated_server) on [Kubernetes](https://kubernetes.io).

_Agones, is derived from the Greek word agōn which roughly translates to “contest”, “competition at games” and “gathering”.
([source](https://www.merriam-webster.com/dictionary/agones))_

## Disclaimer
This software is currently alpha, and subject to change. Not to be used in production systems.

## Major Features
- Be able to define a `GameServer` within Kubernetes - either through yaml or via the API
- Manage GameServer lifecycles - including health checking and connection information.
- Client SDKs for integration with dedicated game servers to work with Agones.

## Why does this project exist?
For more details on why this project was written, read the
[announcement blog post](https://cloudplatform.googleblog.com/2018/03/introducing-Agones-open-source-multiplayer-dedicated-game-server-hosting-built-on-Kubernetes.html).

## Requirements
- Kubernetes cluster version 1.9+
    - [Minikube](https://github.com/kubernetes/minikube), [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/) and [Azure Kubernetes Service](https://azure.microsoft.com/en-us/services/kubernetes-service/) have been tested
    - If you are creating and managing your own Kubernetes cluster, the
    [MutatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#mutatingadmissionwebhook-beta-in-19), and
    [ValidatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#validatingadmissionwebhook-alpha-in-18-beta-in-19)
    admission controllers are required.
    We also recommend following the
    [recommended set of admission controllers](https://kubernetes.io/docs/admin/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use).
- Firewall access for the range of ports that Game Servers can be connected to in the cluster.
- Game Servers must have the [project SDK](sdks) integrated, to manage Game Server state, health checking, etc.

## Installation

Follow [these instructions](install/README.md) to create a cluster on Google Kubernetes Engine (GKE), Minikube or Azure Kubernetes Service (AKS), and install Agones.

## Usage

Documentation and usage guides on how to develop and host dedicated game servers on top of Agones.

### Quickstarts:
 - [Create a Game Server](./docs/create_gameserver.md)
 - [Create a Game Server Fleet](./docs/create_fleet.md)
 - [Create a Fleet Autoscaler](./docs/create_fleetautoscaler.md)
 - [Edit Your First Game Server (Go)](./docs/edit_first_game_server.md)

### Guides
 - [Integrating the Game Server SDK](sdks)
 - [GameServer Health Checking](./docs/health_checking.md)
 - [Accessing Agones via the Kubernetes API](./docs/access_api.md)
 - [Troubleshooting](./docs/troubleshooting.md)

### Tutorials
 - [Create an Allocator Service (Go)](./docs/create_allocator_service.md) - Learn to programmatically access Agones via the API

### Reference
- [Game Server Specification](./docs/gameserver_spec.md)
- [Fleet Specification](./docs/fleet_spec.md)
- [Fleet Autoscaler Specification](./docs/fleetautoscaler_spec.md)

### Examples
- [Full GameServer Configuration](./examples/gameserver.yaml)
- [Full Fleet Configuration](./examples/fleet.yaml)
- [Full Fleet Allocation Configuration](./examples/fleetallocation.yaml)
- [Simple UDP](./examples/simple-udp) (Go) - simple server and client that send UDP packets back and forth.
- [CPP Simple](./examples/cpp-simple) (C++) - C++ example that starts up, stays healthy and then shuts down after 60 seconds.
- [Xonotic](./examples/xonotic) - Wraps the SDK around the open source FPS game [Xonotic](http://www.xonotic.org) and hosts it on Agones.

## Get involved

- [Slack](https://join.slack.com/t/agones/shared_invite/enQtMzE5NTE0NzkyOTk1LWQ2ZmY1Mjc4ZDQ4NDJhOGYxYTY2NTY0NjUwNjliYzVhMWFjYjMxM2RlMjg3NGU0M2E0YTYzNDIxNDMyZGNjMjU)
- [Twitter](https://twitter.com/agonesdev)
- [Mailing List](https://groups.google.com/forum/#!forum/agones-discuss)

## Code of Conduct

Participation in this project comes under the [Contributor Covenant Code of Conduct](code-of-conduct.md)

## Development and Contribution

Please read the [contributing](CONTRIBUTING.md) guide for directions on submitting Pull Requests to Agones.

See the [Developing, Testing and Building Agones](build/README.md) documentation for developing, testing and building Agones from source.

The [Release Process](docs/governance/release_process.md) documentation displays the project's upcoming release calendar and release process.

Agones is in active development - we would love your help in shaping its future!

## This all sounds great, but can you explain Docker and/or Kubernetes to me?

### Docker
- [Docker's official "Getting Started" guide](https://docs.docker.com/get-started/)
- [Katacoda's free, interactive Docker course](https://www.katacoda.com/courses/docker)

### Kubernetes
- [You should totally read this comic, and interactive tutorial](https://cloud.google.com/kubernetes-engine/kubernetes-comic/)
- [Katacoda's free, interactive Kubernetes course](https://www.katacoda.com/courses/kubernetes)

## Licence

Apache 2.0
