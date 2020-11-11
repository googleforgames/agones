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

## Testing

```bash
# Get the GameServer definition
$ kubectl get gs
NAME               STATE       ADDRESS         PORT   NODE                                        AGE
simple-tcp-6mwwn   Unhealthy   34.105.58.178   7685   gke-xonotic-xonotic-windows-8bb8ecec-qpjg   35m
simple-tcp-6rkk7   Unhealthy   34.105.58.178   7032   gke-xonotic-xonotic-windows-8bb8ecec-qpjg   69m
simple-tcp-b4ccz   Ready       34.105.58.178   7627   gke-xonotic-xonotic-windows-8bb8ecec-qpjg   14m
simple-tcp-d9dfc   Ready       34.83.210.144   7083   gke-xonotic-default-pool-14f8be3d-j2hj      5m41s
simple-tcp-lfns5   Unhealthy   34.105.58.178   7051   gke-xonotic-xonotic-windows-8bb8ecec-qpjg   15m
simple-tcp-rdchv   Ready       34.105.58.178   7654   gke-xonotic-xonotic-windows-8bb8ecec-qpjg   6m36s
simple-tcp-wprpp   Unhealthy   34.105.58.178   7584   gke-xonotic-xonotic-windows-8bb8ecec-qpjg   70m

# Use Netcat
$ nc 34.83.210.144 7083
Hello Agones
ACK: Hello Agones
```
