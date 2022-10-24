---
title: "Frequently Asked Questions"
linkTitle: "FAQ"
weight: 400
date: 2020-04-06
---

## Architecture

### What is the relationship between a Kubernetes Pod and an Agones GameServer?

Agones creates a backing Pod with the appropriate configuration parameters for each GameServer that is configured in
the cluster. They both have the same name if you are ever looking to match one to the other.

### Can I reuse a GameServer for multiple game sessions?

Yes.

Agones is inherently un-opinionated about the lifecycle of your game.  When you call
[SDK.Allocate()]({{< ref "/docs/Guides/Client SDKs/_index.md#allocate" >}}) you are
protecting that GameServer instance from being scaled down for the duration of the Allocation.  Typically, you would
run one game session within a single allocation.  However, you could allocate, and run N sessions on a single
GameServer, and then de-allocate/shutdown at a later time.

### How can I return an `Allocated` GameServer to the `Ready` state?

If you wish to return an `Allocated` GameServer to the `Ready` state, you can use the
[SDK.Ready()]({{< ref "/docs/Guides/Client SDKs/_index.md#ready" >}}) command whenever it
makes sense for your GameServer to return to the pool of potentially Allocatable and/or scaled down GameServers.

Have a look at the integration pattern
["Reusing Allocated GameServers for more than one game session"]({{% ref "/docs/Integration Patterns/reusing-gameservers.md" %}}) 
for more details.

## Integration

### What steps do I need to take to integrate my GameServer?

1. Integrate your game server binary with the [Agones SDK]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}), 
   calling the appropriate [lifecycle event]({{< ref "/docs/Guides/Client SDKs/_index.md#lifecycle-management" >}}) 
   hooks.
1. Containerize your game server binary with [Docker](https://www.docker.com/)
1. Publish your Docker image in a [container registry/repository](https://docs.docker.com/docker-hub/repos/).
1. Create a [gameserver.yaml]({{< ref "/docs/Reference/gameserver.md" >}}) file for your container image.
1. Test your gameserver.yaml file.
1. Consider utilizing [Fleets]({{< ref "/docs/Reference/fleet.md" >}}).
   and [Autoscalers]({{< ref "/docs/Reference/fleetautoscaler.md" >}}) for deploying at scale.

### What are some common patterns for integrating the SDK with a Game Server Binary?

* In-Engine
   * Integrate the SDK directly with the dedicated game server, such that it is part of the same codebase.
* Sidecar
   * Use a Kubernetes [sidecar pattern](https://blog.davemdavis.net/2018/03/13/the-sidecar-pattern/) to run the SDK
     in a separate process that runs alongside your game server binary, and can share the disk and network namespace.
     This game server binary could expose its own API, or write to a shared file, that the sidecar process
     integrates with, and can then communicate back to Agones through the SDK.
* Wrapper
   * Write a process that wraps the game server binary, and intercepts aspects such as the foreground log output, and
     use that information to react and communicate with Agones appropriately.
     This can be particularly useful for legacy game servers or game server binaries wherein you do not have access to
     the original source. You can see this in both the {{< ghlink href="examples/xonotic" >}}Xonotic{{< /ghlink >}} and
     {{< ghlink href="examples/supertuxkart" >}}SuperTuxKart{{< /ghlink >}} examples.  

### What if my engine / language of choice does not have a supported SDK, what can I do?

Either utilise the [REST API]({{< ref "/docs/Guides/Client SDKs/rest.md" >}}), which can be 
[generated from the Swagger specification]({{< ref "/docs/Guides/Client SDKs/rest.md#generating-clients" >}}), 
or [generate your own gRPC client from the proto file]({{< ref "/docs/Guides/Client SDKs/_index.md" >}}). 

Game Server SDKs are a thin wrapper around either REST or gRPC clients, depending on language or platform, and can be
used as examples.

### How can I pass data to my Game Server binary on Allocation?

A `GameServerAllocation` has a [spec.metadata section]({{< ref "/docs/Reference/gameserverallocation.md" >}}), 
that will apply any configured [Labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
and/or [Annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) to a requested
GameServer at Allocation time.

The game server binary can watch for the state change to `Allocated`, as well as changes to the GameServer metadata,
through [SDK.WatchGameServer()]({{< ref "/docs/Guides/Client SDKs/_index.md#watchgameserverfunctiongameserver" >}}).

Combining these two features allows you to pass information such as map data, gameplay metadata and more to a game
server binary at Allocation time, through Agones functionality.

Do note, that if you wish to have either the labels or annotations on the `GameServer` that are set via a  
`GameServerAllocation` to be editable by the game server binary with the Agones SDK, the label key will need to 
be prefixed with `agones.dev/sdk-`.
See [SDK.SetLabel()]({{< ref "/docs/Guides/Client SDKs/_index.md#setlabelkey-value" >}})
and [SDK.SetAnnotation()]({{< ref "/docs/Guides/Client SDKs/_index.md#setannotationkey-value" >}}) for more information.

### How can I expose information from my game server binary to an external service?

The Agones game server SDK allows you to set custom 
[Labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/) 
and [Annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) through 
the [SDK.SetLabel()]({{< ref "/docs/Guides/Client SDKs/_index.md#setlabelkey-value" >}}) 
and [SDK.SetAnnotation()]({{< ref "/docs/Guides/Client SDKs/_index.md#setannotationkey-value" >}}) functionality
respectively.

This information is then queryable via the [Kubernetes API]({{< ref "/docs/Guides/access-api.md" >}}), 
and can be used for game specific, custom integrations. 

### If my game server requires more states than what Agones provides (e.g. Ready, Allocated, Shutdown, etc), can I add my own? 

If you want to track custom game server states, then you can utilise the game server client SDK
[SDK.SetLabel()]({{< ref "/docs/Guides/Client SDKs/_index.md#setlabelkey-value" >}}) 
and [SDK.SetAnnotation()]({{< ref "/docs/Guides/Client SDKs/_index.md#setannotationkey-value" >}}) functionality to
expose these custom states to outside systems via your own labels and annotations.

This information is then queryable via the [Kubernetes API]({{< ref "/docs/Guides/access-api.md" >}}), and 
can be used for game specific state integrations with systems like matchmakers and more.

Custom labels could also potentially be utilised with [GameServerAllocation required and/or preferred label
selectors]({{< ref "/docs/Reference/gameserverallocation.md" >}}), to further refine `Ready` GameServer
selection on Allocation.

## Scaling

### How large can an Agones cluster be? / How many GameServers can be supported in a single cluster?

The answer to this question is "it depends" 😁.

As a rule of thumb, we recommend clusters no larger than 500 nodes, based on production workloads.

That being said, this is highly dependent on Kubernetes hosting platform, control plane resources, node resources, 
requirements of your game server, game server session length, node spin up time, etc, and therefore you 
should run your own load tests against your hosting provider to determine the optimal cluster size for your game.

We recommend running multiple clusters for your production GameServer workloads, to spread the load and
provide extra redundancy across your entire game server fleet.

## Network

### How are IP addresses allocated to GameServers?

Each `GameServer` inherits the IP Address of the Node on which it resides. If it can find an `ExternalIP` address on
the Node (which it should if it's a publicly addressable Node), that it utilised, otherwise it falls back to using the
`InternalIP` address. 

### How do I use the DNS name of the Node?
  
If the Kubernetes nodes have an `ExternalDNS` record, then it will be utilised as the `GameServer` address 
preferentially over the `ExternalIP` node record.

### How is traffic routed from the allocated Port to the GameServer container?

Traffic is routed to the GameServer Container utilising the `hostPort` field on a 
[Pod's Container specification]({{< k8s-api href="#containerport-v1-core" >}}).

This opens a port on the host Node and routes traffic to the container 
via [iptables](https://en.wikipedia.org/wiki/Iptables) or 
[ipvs](https://kubernetes.io/blog/2018/07/09/ipvs-based-in-cluster-load-balancing-deep-dive/), depending on host
provider and/or network overlay.

In worst case scenarios this routing can add an extra 0.5ms latency to UDP packets, but that is extremely rare.

#### Why did you use hostPort and not hostNetwork for your networking?

The decision was made not to use `hostNetwork`, as the benefits of having isolated network namespaces between
game server processes give us the ability to run
[sidecar containers](https://blog.davemdavis.net/2018/03/13/the-sidecar-pattern/), and provides an extra layer of
security to each game server process.

## Performance

### How big an image can I use for my GameServer?

We routinely see users running container images that are multiple GB in size.

The only downside to larger images, is that they can take longer to first load on a Kubernetes node, but that can be
managed by your 
[Fleet]({{< ref "/docs/Reference/fleet.md" >}}) and 
[Fleet Autoscaling]({{< ref "/docs/Reference/fleetautoscaler.md" >}})
configuration to ensure this load time is taken into account on a new Node's container initial load.

### How quickly can Agones spin up new GameServer instances?

When running Agones on GKE, we have verified that an Agones cluster can start up 
to 10,000 GameServer instances per minute (not including node creation).

This number could vary depending on the underlying scaling capabilities
of your cloud provider, Kubernetes cluster configuration, and your GameServer Ready startup time, and
therefore we recommend you always run your own load tests for your specific game and game server containers.

## Operating Systems

### Are Windows Container game servers supported by Agones?

As of Kubernetes 1.14, Windows Container support
[has been released as GA](https://kubernetes.io/blog/2019/03/25/kubernetes-1-14-release-announcement/).

That being said, Agones has yet to be tested with Windows Nodes and work on this feature has not been started.

If you are interested in this feature and/or contributing, please add a comment to the 
[Running windows game server](https://github.com/googleforgames/agones/issues/54) ticket.

## Ecosystem

### Is there an example of Agones and Open Match working together? 

Yes! There are several! Check out both our
[official]({{% ref "/docs/Examples/_index.md#integration-with-open-match" %}}) 
and [third party]({{% ref "/docs/Third Party Content/examples.md#integration-with-open-match" %}}) examples!
