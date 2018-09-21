# Simple UDP Server

A very simple UDP logging server, for the purposes of demoing and testing
running a UDP based server on Agones.

## Server
Starts a server on port `7654` by default. Can be overwritten by `PORT` env var or `port` flag.

When it receives a text packet, it will send back "ACK:<text content>" as an echo. 

If it receives the text "EXIT", then it will `sys.Exit(0)`

To learn how to deploy your edited version of go server to gcp, please check out this link: [Edit Your First Game Server (Go)](../../../master/docs/edit_first_game_server.md),
or also look at the [`Makefile`](./Makefile).
