# Simple Game Server

A very simple logging server, for the purposes of demoing and testing
running UDP and/or TCP based server on Agones.

## Server
Starts a server on port `7654` by default. Can be overwritten by `PORT` env var or `port` flag.

By default listens on UDP. Set `UDP=FALSE` env var or `-udp=false` flag to turn UDP off.

To enable TCP, set `TCP=TRUE` env var or `-tcp` flag. Must have at least TCP or UDP enabled.

When it receives a text packet, it will send back "ACK:<text content>" for UDP as an echo or "ACK TCP:<text content>" for TCP. 

If it receives the text "EXIT", then it will `sys.Exit(0)`

To learn how to deploy your edited version of go server to gcp, please check out this link: [Edit Your First Game Server (Go)](https://agones.dev/site/docs/getting-started/edit-first-gameserver-go/),
or also look at the [`Makefile`](./Makefile).
