# Troubleshooting

Troubleshooting guides and steps.

Table of Contents
=================

* [How do I see the logs for Agones?](#how-do-i-see-the-logs-for-agones)
* [I uninstalled Agones before deleted all my GameServers and now they won't delete](#i-uninstalled-agones-before-deleted-all-my-gameservers-and-now-they-wont-delete)
* [I'm getting Forbidden errors when trying to install Agones](#im-getting-forbidden-errors-when-trying-to-install-agones)

## How do I see the logs for Agones?

If something is going wrong, and you want to see the logs for Agones, there are potentially two places you will want to
check:

1. The controller: assuming you installed Agones in the `agones-system` namespace, you will find that there
is a single pod called `agones-controller-<hash>` (where hash is the unique code that Kubernetes generates) 
that exists there, that you can get the logs from. This is the main
controller for Agones, and should be the first place to check when things go wrong.  
   
   To get the logs from this controller run:   
   `kubectl logs --namespace=agones-system agones-controller-<hash>`   
2. The sdk server sidecar: Agones runs a small [gRPC](https://grpc.io/) + http server for the SDK in a container in the
same network namespace as the game server container to connect to via the SDK.  
The logs from this SDk server are also useful for tracking down issues, especially if you are having trouble with a
particular `GameServer`.   
   1. To find the `Pod` for the `GameServer` look for the pod with a name that is prefixed with the name of the 
   owning `GameServer`. For example if you have a `GameServer` named `simple-udp`, it's pod could potentially be named
   `simple-udp-dnbwj`.
   2. To get the logs from that `Pod`, we need to specify that we want the logs from the `agones-gameserver-sidecar`
   container. To do that, run the following:   
   `kubectl logs simple-udp-dnbwj -c agones-gameserver-sidecar`

Agones uses JSON structured logging, therefore errors will be visible through the `"severity":"info"` key and value.       

## I uninstalled Agones before deleted all my `GameServers` and now they won't delete

Agones `GameServers` use [Finalizers](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers)
to manage garbage collection of the `GameServers`. This means that if the Agones controller 
doesn't remove the finalizer for you (i.e. if it has been uninstalled),  it can be tricky to remove them all.

Thankfully, if we create a patch to remove the finalizers from all GameServers, we can delete them with impunity.

A quick one liner to do this:

`kubectl get gameserver -o name | xargs -n1 -P1 -I{} kubectl patch {} --type=merge -p '{"metadata": {"finalizers": []}}'`

Once this is done, you can `kubectl delete gs --all` and clean everything up (if it's not gone already).

## I'm getting Forbidden errors when trying to install Agones

Some troubleshooting steps:

1. Run `kubectl describe clusterrolebinding cluster-admin-binding` and make sure your email is in there. This may be
_case sensitive_ so you may need to compare it to the case you used.
1. In the [GKE tutorial](../install/README.md#enabling-creation-of-rbac-resources) `gcloud config get-value accounts` 
will return a lowercase email address, so if you are using a CamelCase email, you may want to type that in manually.
