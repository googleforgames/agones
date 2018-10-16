# Install and configure Agones on Kubernetes

In this quickstart, we will create a Kubernetes cluster, and populate it with the resource types that power Agones.

# Table of contents

1. [Setting up a Google Kubernetes Engine (GKE) cluster](#setting-up-a-google-kubernetes-engine-gke-cluster)
   1. [Before you begin](#before-you-begin)
   1. [Choosing a shell](#choosing-a-shell)
      1. [Cloud shell](#cloud-shell)
      1. [Local shell](#local-shell)
   1. [Creating the cluster](#creating-the-cluster)
      1. [Creating the firewall](#creating-the-firewall)
1. [Setting up a Minikube cluster](#setting-up-a-minikube-cluster)
   1. [Installing Minikube](#installing-minikube)
   1. [Creating an agones profile](#creating-an-agones-profile)
   1. [Starting Minikube](#starting-minikube)
1. [Setting up an Amazon Web Services EKS cluster](#setting-up-an-amazon-web-services-eks-cluster)
    1. [Create EKS Instance](#create-eks-instance)
    1. [Ensure VPC CNI 1.2 is Running](#ensure-vpc-cni-12-is-running)
    1. [Follow Normal Instructions to Install](#follow-normal-instructions-to-install)
1. [Setting up an Azure Kubernetes Service (AKS) cluster](#setting-up-an-azure-kubernetes-service-aks-cluster)
    1. [Choosing your shell](#choosing-your-shell)
    1. [Creating the AKS cluster](#creating-the-aks-cluster)
    1. [Allowing UDP traffic](#allowing-udp-traffic)
    1. [Creating and assigning Public IPs to Nodes](#creating-and-assigning-public-ips-to-nodes)
1. [Enabling creation of RBAC resources](#enabling-creation-of-rbac-resources)
1. [Installing Agones](#installing-agones)
   1. [Install with yaml](#install-with-yaml)
   1. [Install using Helm](#install-using-helm)
   1. [Confirming Agones started successfully](#confirming-agones-started-successfully)
1. [What's next](#whats-next)

# Setting up a Google Kubernetes Engine (GKE) cluster

Follow these steps to create a cluster and install Agones directly on Google Kubernetes Engine (GKE).

## Before you begin

Take the following steps to enable the Kubernetes Engine API:

1. Visit the [Kubernetes Engine][kubernetes] page in the Google Cloud Platform Console.
1. Create or select a project.
1. Wait for the API and related services to be enabled. This can take several minutes.
1. [Enable billing][billing] for your project.
   * If you are not an existing GCP user, you may be able to enroll for a $300 US [Free Trial][trial] credit.

[kubernetes]: https://console.cloud.google.com/kubernetes/list
[billing]: https://support.google.com/cloud/answer/6293499#enable-billing
[trial]: https://cloud.google.com/free/

## Choosing a shell

To complete this quickstart, we can use either [Google Cloud Shell][cloud-shell] or a local shell.

Google Cloud Shell is a shell environment for managing resources hosted on Google Cloud Platform (GCP). Cloud Shell comes preinstalled with the [gcloud][gcloud] and [kubectl][kubectl] command-line tools. `gcloud` provides the primary command-line interface for GCP, and `kubectl` provides the command-line interface for running commands against Kubernetes clusters.

If you prefer using your local shell, you must install the gcloud and kubectl command-line tools in your environment.

[cloud-shell]: https://cloud.google.com/shell/
[gcloud]: https://cloud.google.com/sdk/gcloud/
[kubectl]: https://kubernetes.io/docs/user-guide/kubectl-overview/

### Cloud shell

To launch Cloud Shell, perform the following steps:

1. Go to [Google Cloud Platform Console][cloud]
1. From the top-right corner of the console, click the **Activate Google Cloud Shell** button: ![cloud shell](/docs/cloud-shell.png?raw=true)
1. A Cloud Shell session opens inside a frame at the bottom of the console. Use this shell to run `gcloud` and `kubectl` commands.
1. Set a compute zone in your geographical region with the following command. The compute zone will be something like `us-west1-a`. A full list can be found [here][zones].
   ```bash
   gcloud config set compute/zone [COMPUTE_ZONE]
   ```

[cloud]: https://console.cloud.google.com/home/dashboard
[zones]: https://cloud.google.com/compute/docs/regions-zones/#available

### Local shell

To install `gcloud` and `kubectl`, perform the following steps:

1. [Install the Google Cloud SDK][gcloud-install], which includes the `gcloud` command-line tool.
1. Initialize some default configuration by running the following command.
   * When asked `Do you want to configure a default Compute Region and Zone? (Y/n)?`, enter `Y` and choose a zone in your geographical region of choice.
   ```bash
   gcloud init
   ```
1. Install the `kubectl` command-line tool by running the following command:
   ```bash
   gcloud components install kubectl
   ```

[gcloud-install]: https://cloud.google.com/sdk/docs/quickstarts

## Creating the cluster

A [cluster][cluster] consists of at least one *cluster master* machine and multiple worker machines called *nodes*: [Compute Engine virtual machine][vms] instances that run the Kubernetes processes necessary to make them part of the cluster.

```bash
gcloud container clusters create [CLUSTER_NAME] --cluster-version=1.10 \
  --no-enable-legacy-authorization \
  --tags=game-server \
  --enable-basic-auth \
  --password=supersecretpassword \
  --scopes=https://www.googleapis.com/auth/devstorage.read_only,compute-rw,cloud-platform \
  --num-nodes=3 \
  --machine-type=n1-standard-1
```

Flag explanations:

* cluster-version: Agones requires Kubernetes version 1.9+. Once the default version reaches 1.9, this will no longer be necessary.
* no-enable-legacy-authorization: This enables RBAC, the authorization scheme used by Agones to control access to resources.
* tags: Defines the tags that will be attached to new nodes in the cluster. This is to grant access through ports via the firewall created in the next step.
* enable-basic-auth/password: Sets the master auth scheme for interacting with the cluster.
* scopes: Defines the Oauth scopes required by the nodes.
* num-nodes: The number of nodes to be created in each of the cluster's zones. Default: 3
* machine-type: The type of machine to use for nodes. Default: n1-standard-1.

Finally, let's tell `gcloud` that we are speaking with this cluster, and get auth credentials for `kubectl` to use.

```bash
gcloud config set container/cluster [CLUSTER_NAME]
gcloud container clusters get-credentials [CLUSTER_NAME]
```

[cluster]: https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture
[vms]: https://cloud.google.com/compute/docs/instances/

### Creating the firewall

We need a firewall to allow UDP traffic to nodes tagged as `game-server` via ports 7000-8000.

```bash
gcloud compute firewall-rules create game-server-firewall \
  --allow udp:7000-8000 \
  --target-tags game-server \
  --description "Firewall to allow game server udp traffic"
```

Continue to [Enabling creation of RBAC resources](#enabling-creation-of-rbac-resources)

# Setting up a Minikube cluster

This will setup a [Minikube](https://github.com/kubernetes/minikube) cluster, running on an `agones` profile.

## Installing Minikube

First, [install Minikube][minikube], which may also require you to install
a virtualisation solution, such as [VirtualBox][vb] as well.

[minikube]: https://github.com/kubernetes/minikube#installation
[vb]: https://www.virtualbox.org

> We recommend installing version [0.29.0 of minikube](https://github.com/kubernetes/minikube/releases/tag/v0.29.0).

## Creating an `agones` profile

Let's use a minikube profile for `agones`.

```bash
minikube profile agones
```

## Starting Minikube

The following command starts a local minikube cluster via virtualbox - but this can be
replaced by a [vm-driver](https://github.com/kubernetes/minikube#requirements) of your choice.

```bash
minikube start --kubernetes-version v1.10.0 --vm-driver virtualbox \
		--extra-config=apiserver.admission-control=NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota \
		--extra-config=apiserver.authorization-mode=RBAC
```

> the --bootstrapper=localkube is required since we aren't using the `default` profile. ([bug](https://github.com/kubernetes/minikube/issues/2717))

# Setting up an Amazon Web Services EKS cluster

## Create EKS Instance

Create your EKS instance using the [Getting Started Guide](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html).

## Ensure VPC CNI 1.2 is Running

EKS does not use the normal Kubernetes networking since it is [incompatible with Amazon VPC networking](https://www.contino.io/insights/kubernetes-is-hard-why-eks-makes-it-easier-for-network-and-security-architects). 

In a console, run this command to get your current cni version

```bash
kubectl describe daemonset aws-node --namespace kube-system | grep Image | cut -d "/" -f 2
```
Output should be `amazon-k8s-cni:1.2.0` or newer. To upgrade to version 1.2, run the following command.

```bash
kubectl apply -f https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/master/config/v1.2/aws-k8s-cni.yaml
```

## Follow Normal Instructions to Install

Continue to [Installing Agones](#installing-agones).

# Setting up an Azure Kubernetes Service (AKS) Cluster

Follow these steps to create a cluster and install Agones directly on [Azure Kubernetes Service (AKS) ](https://docs.microsoft.com/azure/aks/).

## Choosing your shell

You can use either [Azure Cloud Shell](https://docs.microsoft.com/azure/cloud-shell/overview) or install the [Azure CLI](https://docs.microsoft.com/cli/azure/?view=azure-cli-latest) on your local shell in order to install AKS in your own Azure subscription. Cloud Shell comes preinstalled with `az` and `kubectl` utilities whereas you need to install them locally if you want to use your local shell. If you use Windows 10, you can use the [WIndows Subsystem for Windows](https://docs.microsoft.com/windows/wsl/install-win10) as well.

## Creating the AKS cluster

If you are using Azure CLI from your local shell, you need to login to your Azure account by executing the `az login` command and following the login procedure.

Here are the steps you need to follow to create a new AKS cluster (additional instructions and clarifications are listed [here](https://docs.microsoft.com/azure/aks/kubernetes-walkthrough)): 

```bash
# Declare necessary variables, modify them according to your needs
AKS_RESOURCE_GROUP=akstestrg     # Name of the resource group your AKS cluster will be created in
AKS_NAME=akstest     # Name of your AKS cluster
AKS_LOCATION=westeurope     # Azure region in which you'll deploy your AKS cluster

# Create the Resource Group where your AKS resource will be installed
az group create --name $AKS_RESOURCE_GROUP --location $AKS_LOCATION

# Create the AKS cluster - this might take some time. Type 'az aks create -h' to see all available options
# The following command will create a single Node AKS cluster. Node size is Standard A1 v1 and Kubernetes version is 1.9.6. Plus, SSH keys will be generated for you, use --ssh-key-value to provide your values
az aks create --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME --node-count 1 --generate-ssh-keys --node-vm-size Standard_A1_v2 --kubernetes-version 1.9.6 --enable-rbac

# Install kubectl
sudo az aks install-cli

# Get credentials for your new AKS cluster
az aks get-credentials --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME
```

Alternatively, you can use the [Azure Portal](https://portal.azure.com) to create a new AKS cluster [(instructions)](https://docs.microsoft.com/azure/aks/kubernetes-walkthrough-portal).

### Allowing UDP traffic

For Agones to work correctly, we need to allow UDP traffic to pass through to our AKS cluster. To achieve this, we must update the NSG (Network Security Group) with the proper rule. A simple way to do that is:

* Login to the Azure Portal
* Find the resource group where the AKS resources are kept, which should have a name like `MC_resourceGroupName_AKSName_westeurope`. Alternative, you can type `az resource show --namespace Microsoft.ContainerService --resource-type managedClusters -g $AKS_RESOURCE_GROUP -n $AKS_NAME -o json | jq .properties.nodeResourceGroup`
* Find the Network Security Group object, which should have a name like `aks-agentpool-********-nsg`
* Select **Inbound Security Rules**
* Select **Add** to create a new Rule with **UDP** as the protocol and **7000-8000** as the Destination Port Ranges. Pick a proper name and leave everything else at their default values

Alternatively, you can use the following command, after modifying the `RESOURCE_GROUP_WITH_AKS_RESOURCES` and `NSG_NAME` values:

```bash
az network nsg rule create \
  --resource-group RESOURCE_GROUP_WITH_AKS_RESOURCES \
  --nsg-name NSG_NAME \
  --name AgonesUDP \
  --access Allow \
  --protocol Udp \
  --direction Inbound \
  --priority 520 \
  --source-port-range "*" \
  --destination-port-range 7000-8000
  ```

### Creating and assigning Public IPs to Nodes

Nodes in AKS don't get a Public IP by default. To assign a Public IP to a Node, find the Resource Group where the AKS resources are installed on the [portal](https://portal.azure.com) (it should have a name like `MC_resourceGroupName_AKSName_westeurope`). Then, you can follow the instructions [here](https://blogs.technet.microsoft.com/srinathv/2018/02/07/how-to-add-a-public-ip-address-to-azure-vm-for-vm-failed-over-using-asr/) to create a new Public IP and assign it to the Node/VM. For more information on Public IPs for VM NICs, see [this document](https://docs.microsoft.com/azure/virtual-network/virtual-network-network-interface-addresses). If you are looking for an automated way to create and assign Public IPs for your AKS Nodes, check [this project](https://github.com/dgkanatsios/AksNodePublicIPController).

Continue to [Installing Agones](#installing-agones).

# Enabling creation of RBAC resources

To install Agones, a service account needs permission to create some special RBAC resource types.

```bash
# Kubernetes Engine
kubectl create clusterrolebinding cluster-admin-binding \
  --clusterrole cluster-admin --user `gcloud config get-value account`
# Minikube
kubectl create clusterrolebinding cluster-admin-binding \
  --clusterrole=cluster-admin --serviceaccount=kube-system:default
```

# Installing Agones

This will install Agones in your cluster.

## Install with YAML

We can install Agones to the cluster using the
[install.yaml](https://github.com/GoogleCloudPlatform/agones/blob/release-0.5.0/install/yaml/install.yaml) file.

```bash
kubectl create namespace agones-system
kubectl apply -f https://github.com/GoogleCloudPlatform/agones/raw/release-0.5.0/install/yaml/install.yaml
```

You can also find the install.yaml in the latest `agones-install` zip from the [releases](https://github.com/GoogleCloudPlatform/agones/releases) archive.

> Note: Installing Agones with the `install.yaml` will setup the TLS certificates stored in this repository for securing
> kubernetes webhooks communication. If you want to generate new certificates or use your own,
> we recommend using the helm installation.

## Install using Helm

Also, we can install Agones using [Helm][helm] package manager. If you want more details and configuration
options see the [Helm installation guide for Agones][agones-install-guide]

[helm]: https://docs.helm.sh
[agones-install-guide]: helm/README.md

## Confirming Agones started successfully

To confirm Agones is up and running, run the following command:

```bash
kubectl describe --namespace agones-system pods
```

It should describe the single pod created in the `agones-system` namespace, with no error messages or status. The `Conditions` section should look like this:

```
Conditions:
  Type           Status
  Initialized    True
  Ready          True
  PodScheduled   True
```

That's it! This creates the [Custom Resource Definitions][crds] that power Agones and allows us to define resources of type `GameServer`.

[crds]: https://kubernetes.io/docs/concepts/api-extension/custom-resources/

# What's next

* Go through the [Create a Game Server Quickstart][quickstart]

[quickstart]: /docs/create_gameserver.md
