# Agon

Agon is a library for running dedicated game servers on [Kubernetes](https://kubernetes.io).

## Disclaimer
This software is currently alpha, and subject to change. Not to be used in production systems.

## Major Features
- Be able to define a `GameServer` within Kubernetes - either through yaml or the via API
- Manage GameServer lifecycles - including health checking and connection information.
- Client SDKs for integration with dedicated game servers to work with Agon.

## Requirements
- Requires a Kubernetes cluster of version 1.8+
- Open the firewall access for the range of ports that Game Servers can be connected to in the cluster.
- Game Servers must have the [project SDK](sdks) integrated, to manage Game Server state, health checking, etc.

## Installation
`kubectl apply -f install.yaml`

If you are running your own Docker repository or want to install a specific version, make a local copy of install.yaml
and edit to match your settings.

_Note:_ There has yet to be a release of Agon, so you will need to edit the `install.yaml` to specify a 
development release or [build from source](build/README.md) 

## Usage

Documentation and usage guides on how to develop and host dedicated game servers on top of Agon.

More documentation forthcoming.

### Quickstarts: 
 - Create a Game Server (forthcoming) 

### Guides
 - Integrating the C++ SDK (forthcoming)
 - [GameServer Health Checking](./docs/health_checking.md)

### Reference
- [SDK](sdks)

### Examples
- [Full GameServer Configuration](./examples/gameserver.yaml)
- [Simple UDP](./examples/simple-udp) (Go) - simple server and client that send UDP packets back and forth.
- [CPP Simple](./examples/cpp-simple) (C++) - C++ example that starts up, stays healthy and then shuts down after 60 seconds.
- [Xonotic](./examples/xonotic) - Wraps the SDK around the open source FPS game [Xonotic](http://www.xonotic.org) and hosts it on Agon. 
 
## Development and Contribution
See the tools in the [build](build/README.md) directory for testing and building Agon from source.

## Licence

Apache 2.0