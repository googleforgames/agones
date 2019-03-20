---
title: "Install Agones using Click to Deploy"
linkTitle: "Install with Click to Deploy"
weight: 4
description: >
  Use Click to Deploy on the Google Cloud Platform Marketplace to try out Agones in a matter of minutes.

---

## Setting up Agones using Click to Deploy

Agones through Click to Deploy allows you to quickly deploy a testing Agones cluster.
We recommend this deployment exclusively for testing, and not production. Once you are
satisfied with your test, please follow our [in-depth guide][agones-install-guide]
to setup your production cluster.

[agones-install-guide]: {{< relref "_index.md" >}}

## Access the application and follow the steps below

Please visit the [Agones Click to Deploy][agones-click-to-deploy] application in the Google Cloud Platform Marketplace.

1. Click ![configure](../configure.png)
1. Either select an existing Google Cloud Platform project or create a new one using the button
![cloud shell](../add-project.png)
  * This is a simplified setup that doesn't enforce all security measures best practices, and as such, **we recommend against using any existing project that is hosting production assets**.
1. You may be asked to [enable billing][billing] for your project, if it is not linked to a billing account yet.
  * If you are not an existing GCP user, you may be able to **enroll for a $300 US [Free Trial][trial] credit**.
1. Either select an existing GKE cluster or create a new one in the region that you would like to run your game servers.
  * For an existing cluster, make sure it **meets [standard requirements][cluster-requirements] for Agones**.
1. Customize the deployment parameters:

    ![namespace](../namespace.png)
    : Kubernetes namespace where Agones and your game servers will be running. Leave it as ``default`` to allows you to avoid having to keep typing extra parameters to ``kubectl`` during testing.
    <hr>

    ![app-instance](../app-instance.png)
    : Kubernetes application name for this Agones installation. Feel free to choose any descriptive identifier for you application deployment.
    <hr>

    ![service-accounts](../service-accounts.png)
    : Leave the 3 service account (bootstrap, controller, sdk) settings **at their default values to ensure a working deployment**.

1. Click deploy and await for the application to be deployed.
1. Follow the section below to create the firewall rules.

[agones-click-to-deploy]: https://console.cloud.google.com/marketplace/details/google/agones
[billing]: https://support.google.com/cloud/answer/6293499#enable-billing
[trial]: https://cloud.google.com/free/
[cluster-requirements]: {{< relref "/docs/Installation/_index.md#creating-the-cluster" >}}

## Creating firewall rules

To ensure connectivity to your game servers, you must create a firewall rule that exposes the range of ports
which can be dynamically allocated to your game servers. On our [in-depth guide][agones-install-guide] we
provide a set a parameters that will only work if you created the cluster with the appropriate network tags
on the backing nodes for your cluster.

Since we expect this cluster to be used for testing and will probably be missing that step, we recommend this
broader firewall rule that will allow access to ports on any hosts in your project. Please do no use this
if you run any production assets in your project.

Open the Cloud Shell for your project by accessing it on the top-right corner of the console. Click the
**Activate Google Cloud Shell** button: ![cloud shell](../cloud-shell.png) and inside the frame that opens at the bottom of the console, use the command below:

```bash
gcloud compute firewall-rules create game-server-firewall \
  --allow udp:7000-8000 \
  --description "Firewall to allow game server udp traffic"
```

You may use this shell to run `gcloud` and `kubectl` commands, which are referenced throughout our guides.

## How to use your Agones cluster

Note that during the installation you selected a namespace to install Agones. If you changed this value to
something else other than ``default``, you may have to pass that parameter to ``kubectl``.

So for example if you chose your namespace to be ``awesome``, then you must also pass the namespace like this:
```bash
kubectl --namespace=awesome get gameservers
```

## What's next

* Go through the [Create a Game Server Quickstart][quickstart]

[quickstart]: {{< ref "/docs/Getting Started/create-gameserver.md" >}}
