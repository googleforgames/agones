# Simple TCP and UDP Server

A very simple game server, for the purposes of testing a TCP and UDP server on the same port

## Server
Starts a server on TCP and UDP port `7654` by default. Can be overwritten by `PORT` env var or `port` flag.

When it receives a TCP text message ending with a newline, it will send back "ACK TCP:<text content>" as an echo. 

When it receives a UDP text packet, it will send back "ACK UDP:<text content>" as an echo. 

If it receives the text "EXIT", then it will `sys.Exit(0)`

## Firewalls

If you plan to access your server remotely, you may need to open up a hole in your
firewall. 

For example, if you created a cluster running on Google Kubernetes Engine following
the installation guide, you can create a firewall rule to allow TCP and UDP traffic to nodes
tagged as game-server via ports 7000-8000.

```bash
gcloud compute firewall-rules create game-server-firewall-tcp \
  --allow tcp:7000-8000,udp7000-8000 \
  --target-tags game-server \
  --description "Firewall to allow game server tcp and udp traffic"
```