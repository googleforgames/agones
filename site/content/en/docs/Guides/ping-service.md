---
title: "Latency Testing with Multiple Clusters"
linkTitle: "Latency Testing Services"
date: 2019-01-03T01:20:30Z
weight: 40
description: >
  When running multiple Agones clusters around the world, you may need to have clients determine which cluster
  to connect to based on latency.  
---

To make latency testing easier, Agones installs with a simple ping service with both HTTP and UDP services that can be called
  for the purpose of timing how long the roundtrip takes for information to be returned from either of these services.

## Installing

By default, Agones installs [Kubernetes Services](https://kubernetes.io/docs/concepts/services-networking/service/) for
both HTTP and the UDP ping endpoints. These can be disabled entirely,
or disabled individually. See the [Helm install guide]({{< ref "/docs/Installation/Install Agones/helm.md" >}}) for the parameters to
 pass through,
as well as configuration options. 

The ping services as all installed under the `agones-system` namespace.

## HTTP Service

This exposes an endpoint that returns a simple text HTTP response on request to the root "/" path. By default this is `ok`, but
it can be configured via the `agones.ping.http.response` parameter. 

This could be useful for providing clusters
with unique lookup names, such that clients are able to identify clusters from their responses.

To lookup the details of this service, run `kubectl describe service agones-ping-http-service --namespace=agones-system`

## UDP Service

The UDP ping service is a rate limited UDP echo service that returns the udp packet that it receives to its designated
sender.

Since UDP sender details can be spoofed, this service is rate limited to 20 requests per second, 
per sender address, per running instance (default is 2).

This rate limit can be raised or lowered via the Helm install parameter `agones.ping.udp.rateLimit`.

UDP packets are also limited to 1024 bytes in size. 

To lookup the details of this service, run `kubectl describe service agones-ping-udp-service --namespace=agones-system`

## Client side tooling

We deliberately didn't provide any game client libraries, as all major languages and engines have capabilities
to send HTTP requests as well as UDP packets.
