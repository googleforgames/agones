# Simple UDP Server

A very simple UDP logging server and client, for the purposes of demoing and testing
running a UDP based server on Agones.

## Server
Starts a server on port `7654` by default. Can be overwritten by `PORT` env var or `port` flag.

When it receives a text packet, it will send back "ACK:<text content>" as an echo. 

If it receives the text "EXIT", then it will `sys.Exit(0)`

## Client
Client will read in from stdin and send each line to the server.

Address defaults to `localhost:7654` but can be changed through the `address` flag.
