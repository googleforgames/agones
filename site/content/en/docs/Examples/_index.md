---
title: "Examples"
date: 2019-01-03T05:05:47Z
description: "List of available code examples"
weight: 120
---

## Configuration files

These are full examples for each of the resource types of Agones

> These examples are for reference only. They are not backed by working images.

- {{< ghlink href="examples/gameserver.yaml" >}}Full GameServer Configuration{{< /ghlink >}}
- {{< ghlink href="examples/fleet.yaml" >}}Full Fleet Configuration{{< /ghlink >}}
- {{< ghlink href="examples/fleetallocation.yaml" >}}Full Fleet Allocation Configuration (deprecated) {{< /ghlink >}}
- {{< ghlink href="examples/gameserverallocation.yaml" >}}Full GameServer Allocation Configuration{{< /ghlink >}}
- {{< ghlink href="examples/fleetautoscaler.yaml" >}}Full Autoscaler Configuration with Buffer Strategy{{< /ghlink >}}
- {{< ghlink href="examples/webhookfleetautoscaler.yaml" >}}Full Autoscaler Configuration with Webhook Strategy{{< /ghlink >}}
- {{< ghlink href="examples/webhookfleetautoscalertls.yaml" >}}Full Autoscaler Configuration with Webhook Strategy + TLS{{< /ghlink >}}

## Game server implementations

These are all examples of simple game server implementations, that integrate the Agones game server SDK. 

- {{< ghlink href="examples/simple-udp" >}}Simple UDP{{< /ghlink >}} (Go) - simple server and client that send UDP packets back and forth.
- {{< ghlink href="examples/cpp-simple" >}}CPP Simple{{< /ghlink >}} (C++) - C++ example that starts up, stays healthy and then shuts down after 60 seconds.
- {{< ghlink href="examples/nodejs-simple" >}}Node.js Simple{{< /ghlink >}} (Node.js) -
  A simple Node.js example that marks itself as ready, sets some labels and then shutsdown.
- {{< ghlink href="examples/rust-simple" >}}Rust Simple{{< /ghlink >}} (Rust) -
  A simple Rust example that marks itself as ready, sets some labels and then shutsdown.
- {{< ghlink href="examples/xonotic" >}}Xonotic{{< /ghlink >}} - Wraps the SDK around the open source FPS game [Xonotic](http://www.xonotic.org) and hosts it on Agones.

## Building on top of Agones

- {{< ghlink href="examples/allocator-service" >}}Allocator Service{{< /ghlink >}} (Go) - 
  This service provides an example of using the 
  [Agones API](https://godoc.org/agones.dev/agones/pkg/client/clientset/versioned/typed/stable/v1alpha1) to allocate a GameServer from a Fleet, 
  and is used in the [Create an Allocator Service (Go)]({{< ref "/docs/Tutorials/allocator-service-go.md" >}}) tutorial.
