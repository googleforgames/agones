---
title: "Third Party Libraries and Tools"
linkTitle: "Libraries and Tools"
date: 2020-05-15
description: "Community contributed libraries and tools on Agones."
weight: 30
---

## Client SDKs

- [Cubxity/AgonesKt](https://github.com/Cubxity/AgonesKt) - Agones Client SDK for **Kotlin**  
- [AndreMicheletti/godot-agones-sdk](https://github.com/AndreMicheletti/godot-agones-sdk) - Agones Client SDK for **Godot Engine**

## Messaging

Libraries or applications that implement messaging systems.

- [Octops/Agones Event Broadcaster](https://github.com/Octops/agones-event-broadcaster) - Broadcast Agones events to the external world
- [Octops/Agones Broadcaster HTTP](https://github.com/Octops/agones-broadcaster-http) - Expose Agones GameServers information via HTTP
- [Octops/Agones Relay HTTP](https://github.com/Octops/agones-relay-http) - Publish Agones GameServers and Fleets details to HTTP endpoints

## Controllers
- [Octops/Game Server Ingress Controller](https://github.com/Octops/gameserver-ingress-controller) - Automatic Ingress configuration for Game Servers managed by Agones
- [Octops/Image Syncer](https://github.com/Octops/octops-image-syncer) - Watch Fleets and pre-pull images of game servers on every node running in the cluster

## Match Making

- [Octops/Agones x Open Match](https://github.com/Octops/agones-discover-openmatch) - How to implement a matchmaking system using Agones and Open Match

## Allocation

- [agones-allocator-client](https://github.com/FairwindsOps/agones-allocator-client) - A client for testing allocation servers.
  Made by [Fairwinds](https://fairwinds.com)
  
## Development Tools

- [Minikube Agones Cluster](https://github.com/comerford/minikube-agones-cluster) - Automates the creation of a complete Kubernetes/Agones Cluster locally, using Xonotic as a sample gameserver. Intended to provide a local environment for developers which approximates a production Agones deployment.
