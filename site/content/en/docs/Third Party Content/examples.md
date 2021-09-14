---
title: "Third Party Examples"
linkTitle: "Examples"
date: 2021-09-09
description: "Community contributed Dedicated Game Server examples on Agones."
weight: 20
---

## Minetest

[Minetest](https://www.minetest.net/) is a free and open-source sandbox video game available for Linux, FreeBSD, Microsoft Windows, MacOS, and Android. Minetest game play is very similar to that of Minecraft. Players explore a blocky 3D world, discover and extract raw materials, craft tools and items, and build structures and landscapes. 

[Minetest server for Agones](https://github.com/paulhkim80/agones-example-minetest) is an example of the Minetest server hosted on Kubernetes using Agones. It wraps the Minetest server with a [Go](https://golang.org) binary, and introspects stdout to provide the event hooks for the SDK integration. The wrapper is from [Xonotic Example](https://github.com/googleforgames/agones/blob/main/examples/xonotic/main.go) with a few changes to look for the Minetest ready output message.  

You will need to download the Minetest client separately to play.
