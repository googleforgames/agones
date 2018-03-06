# Agones
[![GoDoc](https://godoc.org/agones.dev/agones?status.svg)](https://godoc.org/agones.dev/agones)

Agones is a library for hosting, running and scaling [dedicated game servers](https://en.wikipedia.org/wiki/Game_server#Dedicated_server) on [Kubernetes](https://kubernetes.io).

_Agones, is derived from the Greek word agōn which roughly translates to “contest”, “competition at games” and “gathering”.
([source](https://www.merriam-webster.com/dictionary/agones))_

## Disclaimer
This software is currently alpha, and subject to change. Not to be used in production systems.

## Major Features
- Be able to define a `GameServer` within Kubernetes - either through yaml or the via API
- Manage GameServer lifecycles - including health checking and connection information.
- Client SDKs for integration with dedicated game servers to work with Agones.

## Requirements
- Kubernetes cluster version 1.9+
    - [Minikube](https://github.com/kubernetes/minikube) and [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/) have been tested
    - If you are creating and managing your own Kubernetes cluster, the
    [MutatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#mutatingadmissionwebhook-beta-in-19)
    admission controller is required.
    We also recommend following the
    [recommended set of admission controllers](https://kubernetes.io/docs/admin/admission-controllers/#is-there-a-recommended-set-of-admission-controllers-to-use).
- Firewall access for the range of ports that Game Servers can be connected to in the cluster.
- Game Servers must have the [project SDK](sdks) integrated, to manage Game Server state, health checking, etc.

## Installation

Follow [these instructions](./docs/installing_agones.md) to create a cluster on Google Kubernetes Engine (GKE) or Minikube, and install Agones.

## Usage

Documentation and usage guides on how to develop and host dedicated game servers on top of Agones.

### Quickstarts:
 - [Create a Game Server](./docs/create_gameserver.md)

### Guides
 - [Integrating the Game Server SDK](sdks)
 - [GameServer Health Checking](./docs/health_checking.md)
 - [Accessing Agones via the Kubernetes API](./docs/access_api.md)

### Reference
- [Game Server Specification](./docs/gameserver_spec.md)

### Examples
- [Full GameServer Configuration](./examples/gameserver.yaml)
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

See the tools in the [build](build/README.md) directory for testing and building Agones from source.

Agones is in active development - we would love your help in shaping its future!

## Licence

Apache 2.0