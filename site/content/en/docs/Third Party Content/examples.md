---
title: "Third Party Examples"
linkTitle: "Examples"
date: 2021-09-09
description: "Community contributed Dedicated Game Server examples on Agones."
weight: 20
---

## Integrations with other projects

* [Octops/Agones x Open Match](https://github.com/Octops/agones-discover-openmatch) - How to implement a matchmaking 
  system using Agones and Open Match

## Minetest

* [Minetest](https://www.minetest.net/) is a free and open-source sandbox video game available for Linux, FreeBSD, 
Microsoft Windows, MacOS, and Android. Minetest game play is very similar to that of Minecraft. Players explore a blocky 3D world, discover and extract raw materials, craft tools and items, and build structures and landscapes. 
* [Minetest server for Agones](https://github.com/paulhkim80/agones-example-minetest) is an example of the Minetest 
  server hosted on Kubernetes using Agones. It wraps the Minetest server with a [Go](https://golang.org) binary, and introspects stdout to provide the event hooks for the SDK integration. The wrapper is from [Xonotic Example](https://github.com/googleforgames/agones/blob/main/examples/xonotic/main.go) with a few changes to look for the Minetest ready output message.  

You will need to download the Minetest client separately to play.

## Quilkin

* [Quilkin](https://github.com/googleforgames/quilkin) is a non-transparent UDP proxy specifically designed for use with large scale multiplayer dedicated game server deployments, to ensure security, access control, telemetry data, metrics and more.
* [Quilkin with Agones](https://github.com/googleforgames/quilkin/tree/main/examples) is an example of running [Xonotic](https://xonotic.org/) with Quilkin on an Agones cluster, utilising either [the sidecar integration pattern](https://github.com/googleforgames/quilkin/tree/main/examples/agones-xonotic-sidecar) or via the the [Quilkin xDS Agones provider](https://github.com/googleforgames/quilkin/tree/main/examples/agones-xonotic-xds) with a TokenRouter to provide routing and access control to the allocated GameServer instance.

You will need to download the Xonotic client to interact with the demo.

## Shulker

- [Shulker](https://github.com/jeremylvln/Shulker) is a Kubernetes operator for managing complex and dynamic Minecraft
  infrastructure at scale, including game servers and proxies. 
- It builds on top Agones `GameServer` and `Fleet` primitives to provide simplified abstractions specifically tailored
  to orchestrating Minecraft workloads. 

Shulker requires you to have a genuine Minecraft account. You'll need to <a href="https://www.minecraft.net/en-us/article/how-create-minecraft-account" data-proofer-ignore>purchase the game</a>
to test the "Getting Started" example.
