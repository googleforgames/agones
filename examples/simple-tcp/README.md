# Simple TCP Server

A very simple game server, for the purposes of testing a TCP based server on Agones.

## Server
Starts a server on port `7654` by default. Can be overwritten by `PORT` env var or `port` flag.

When it receives a text message ending with a newline, it will send back "ACK:<text content>" as an echo. 

If it receives the text "EXIT", then it will `sys.Exit(0)`

## Firewalls

If you plan to access your server remotely, you may need to open up a hole in your
firewall. 

For example, if you created a cluster running on Google Kubernetes Engine following
the installation guide, you can create a firewall rule to allow TCP traffic to nodes
tagged as game-server via ports 7000-8000.

```bash
gcloud compute firewall-rules create game-server-firewall-tcp \
  --allow tcp:7000-8000 \
  --target-tags game-server \
  --description "Firewall to allow game server udp traffic"
```