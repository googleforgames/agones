# Game Metrics from Dedicated Game Servers
## Summary

Engineers often want to observe "metrics" which can be defined as "raw measurements, generally with the intent to produce continuous summaries of those measurements" from within their Dedicated Game Servers.

A proposal to do this would be to allow the DGS to send a small number (100ish) metrics via the SDK to the running agones sidecar. This requires a simple addition to the sidecar to allow it to expose these metrics via the already used OpenCensus integrations. We can update the SDK to have a metrics sub API (much like alpha and beta stages) to support metric work.

Metrics from dedicated servers often break down into two categories, firtly metrics that Agones already knows about count of players, free capactity etc. and arbitary metrics from within the DGS itself such as frame rate, number of sessions, total rings collected etc.

The main concerns with exposing arbitary metrics is to not expose too many, not impact the running DGS and not to reinvent the wheel but choose the correct technologies to work together.

We could theoretically break this into two proposals:
- expose Agones related metrics (things the sidecar knows about)
- expose DGS metrics (things Agones doesnt need to know about)

### Related Links

- [Agones - Metrics: Players](https://github.com/googleforgames/agones/issues/1035)
- [Agones - Metric: Game Server Frame Rate Over Time](https://github.com/googleforgames/agones/issues/1036)
- [Agones - Pass arbitary metrics from gameserver -> opencensus](https://github.com/googleforgames/agones/issues/1037)

## Goals

- Expose metrics around the current connected players
- Expose arbitary metrics from within the dedicated game server
  - eg. frame rate, number of concurrent sessions, number of failed connections
  - to confirm if this is a desire of the Agones project
- Agnostic to the choice of running engine
  - eg. supports by all engines from C#, Rust through to Unreal and Unity
- Support the basics of Counters, Gauge (UpDownCounter) and labels on these time series
- Minimal impact on the Mem/CPU of the running DGS

## Non-Goals

- Expose metrics around the infrastrucutre usage of the DGS
  - this can be acheived with existing projects
- Reinvent an existing solution
- Support an infinate number of arbitary metrics
- Additional aggregation of metrics
- Storing metrics

## Proposal: Limited number of metrics exposed via sidecar

Initial proposal would be to allow the sidecar to expose metrics itself, initally this would be the specific to the PlayerTracking data such as current number of connected users, free space that can be allocated per DGS and other such metrics that are already known to Agones. This would use the existing OpenCensus (and later OTel) implementation that is being used within Agones to allow these metrics to be scraped by other projects such as Prometheus. Once these metrics are exposed we can simply add another `ServiceMonitor` as is currently done with the controller to communicate with the Prometheus Operator metrics are avaliable to scrape. This would probably require the addition of a agones-sidecar service to route the traffic or defer back to a `PodMonitor`.

With the metrics known to Agones this is simple as state above however, metrics that are not nativley known by Agones (frame rate/sessions etc.) we would need a way to expose these metrics via the SDK to the sidecar. Here, we could simply add a small API that supports the basics of metrics (Counters, Guages, Labels) that sends the data to the sidecar that is exposed in the same way as the Agones native metrics are exposed.

It would be up to the Game Engineers to call this API when data they wish to record changes and would therefore be this would allow them to record the metrics that they are interested in from within the context of their own games.

I would envisage this API being a subset of the Agones SDK such as Alpha and Beta sections currently are. I would also look to limit (could be configurable) the number of metrics a game could send to the sidecar this would reduce the burden on the sidecar and reduce explosion of metrics that are reported.

One advantage of this approach is that we end up storing metrics outside of the DGS and in the sidecar causing much less of a memory impact on the running game. We also create a basic API so that all game running in the Agones project can report metrics in the same way.

The main problem however, is wether this is actually the responsibility of Agones to report on metrics from within a running game server and if so do we then also expose future APIs around events and other aspects of obersvability? I would argue that a mature platform would allow the engineers running their game server to pick up key observability points that they are interested in but could be swayed that this is the responsibility of another project much like CPU/Mem can be obtained directly from kubernetes APIs without the need to intefer with the DGS iteself.

Pro:
- Small extension the Agones SDK
- SDK means all game engines can report in a uniform way
- Native Agones and arbitary metrics can be collected in the same way
- low impact on the memory of the DGS
- Prometheus scraping is already documented and advised as part of the Agones project
  - no need for extra tech as with the alternatives

Con:
- Adding the the SDK surface area
- Push based from the DGS to sidecar (not sure this is a con tbh)

## Alternatives considered
### Expose metrics via logs

This would be my proposal for getting arbitary metrics from a DGS if we were to seperate the two concerns of Agones vs none Agones metrics. In that case I would expose `/metrics` in the sidecar as standard for the data known to Agones (connected users, spaces free etc.).

In this alternative we would use the inbuilt loggers from the engines to log in a given format that then log shippers would be able to turn into metrics. An example of this is the [Prometheus sink](https://vector.dev/docs/reference/sinks/prometheus) for <vector.dev> which would allow you to transform logs and expose them to Prometheus.

The main concern around this is that we probably would not be able to form a decent standard of how to log metrics and would be up to the engineers running the servers and their game teams to discuss the best approach per individual case.

Pros:
- small amount of work in sidecar to expose Agones metrics
- Agones deals with only Agones related metrics

Cons:
- more work for engineers wanting to get metrics from within DGS
- no nice API to program against
- would need to form a standard within each company to send metrics via logs
- some games engines (Unreal) have logs that are designed to be human readable therefore not JSON compatible

### Expose a OpenTelemitry/Prometheus endpoint from within the DGS

This appoach would be a more standardised way of exposing metrics to via a running service and would most probably be supported in engines such as vanilla Rust, Go, Javascript etc. but other games servers Unreal for example does not support the idea of running a web server within the game to expose these metrics.

This also doesnt take into consideration that sidecar has metrics such as the number of connected players, free space on the DGS etc. this means we would probably end up implementing a `/metrics` endpoint within the DGS and its sidecar.

Pros:
- pull based
- supported out of the box in some "engines"
- would leave the engineers to choose technologies (Prom, OTel etc.)
- no work on SDK needed

Cons:
- unsupported in game engines such as Unity and Unreal
- memory impact on the running DGS

### Expose metrics via an agreed file format

- [Prometheus Docs - Text Format](https://prometheus.io/docs/instrumenting/exposition_formats/)

This approach would instead of sending metrics over the wire could instead write metrics to a specific file mounted on the pod in a well known format see the link above. This could then be picked up by something to expose or ship the metrics to a needed place for aggregation.

Pros:
- less memory impact (assumed)

Cons:
- would be down to the engineer to expose it in the needed format
- above link only Prometheus format, may not suite all needs
- sidecar would be exposing a `/metrics` endpoint anyway

