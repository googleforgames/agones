---
title: "Common Integration Patterns"
linkTitle: "Integration Patterns"
weight: 8
date: 2021-07-27
description: >
    Common patterns and integrations of external systems, such as matchmakers, with `GameServer` starting, allocating 
    and shutdown.
---

{{< alert title="Note" color="info" >}}
These examples will use the `GameServerAllocation` resource for convenience, but these same patterns can be applied 
when using the [Allocation Service]({{% ref "/docs/Advanced/allocator-service.md" %}}) instead.
{{< /alert >}}
