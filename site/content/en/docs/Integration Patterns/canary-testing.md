---
title: "Canary Testing a new Fleet"
linkTitle: "Canary Testing"
date: 2021-07-27T23:49:14Z 
weight: 30 
description: >
  Run a small Fleet for the new version of your `GameServer` to ensure it works correctly, before rolling it out to all
  your players.
---

To [canary release/test](https://martinfowler.com/bliki/CanaryRelease.html) a new `Fleet`, 
we can run a small, fixed size `Fleet` of the new version of our GameServer, while also running the current stable 
production version.

`Allocations` can then `prefer` to come from the canary `Fleet`, but if all `GameServers` are already allocated from the 
canary `Fleet`, players will be allocated to the current stable Fleets.

Over time, if the monitoring of those playing on the canary `Fleet` is working as expected, the size of the canary 
`Fleet` can be grown until you feel confident in its stability. 

Once confidence has been achieved, the configuration for stable `Fleet` can be updated to match the canary (usually 
triggering a [rolling update]({{% ref "/docs/Guides/fleet-updates.md#rolling-update-strategy" %}})). The 
canary `Fleet` can then be deleted or updated to a new testing version of the game server process.

<a href="../../../diagrams/canary-testing.puml.png" target="_blank">
<img src="../../../diagrams/canary-testing.puml.png" alt="Canary Fleet Diagram" />
</a>

## Sample `GameServerAllocation`

To ensure we don't have to change the Allocation system every time we have a canary `Fleet`, in this example, we will 
state that in our system, the label `canary: "true"` will be added to any canary `Fleet` in the cluster.   

```yaml
apiVersion: allocation.agones.dev/v1
kind: GameServerAllocation
spec:
  selectors:
    - matchLabels:
        canary: "true"
    - matchLabels:
        agones.dev/fleet: stable
```

The above `Allocation` will then preferentially choose the `Fleet` that has `GameServers` with the label and key 
value of`canary:"true"`, if it exists, and has remaining Ready `GameServers`, and if not, will apply the 
`Allocation` to the `Fleet` named "stable".

## Next Steps

* Read about different [`Fleet` update]({{% ref "/docs/Guides/fleet-updates.md" %}}) options and strategies that are 
  available.
* Have a look at all the options available on a 
  [`GameServerAllocation`]({{% ref "/docs/Reference/gameserverallocation.md" %}}).
* Review the [`Fleet` reference]({{% ref "/docs/Reference/fleet.md" %}}).
