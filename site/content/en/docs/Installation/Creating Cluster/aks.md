---
title: "Azure Kubernetes Service"
linkTitle: "Azure"
weight: 30
description: >
  Follow these steps to create an [Azure Kubernetes Service (AKS) ](https://docs.microsoft.com/azure/aks/) cluster 
  for your Agones install. 
---

## Choosing your shell

You can use either [Azure Cloud Shell](https://docs.microsoft.com/azure/cloud-shell/overview) or install the [Azure CLI](https://docs.microsoft.com/cli/azure/?view=azure-cli-latest) on your local shell in order to install AKS in your own Azure subscription. Cloud Shell comes preinstalled with `az` and `kubectl` utilities whereas you need to install them locally if you want to use your local shell. If you use Windows 10, you can use the [WIndows Subsystem for Windows](https://docs.microsoft.com/windows/wsl/install-win10) as well.

## Creating the AKS cluster

If you are using Azure CLI from your local shell, you need to log in to your Azure account by executing the `az login` command and following the login procedure.

Here are the steps you need to follow to create a new AKS cluster (additional instructions and clarifications are listed [here](https://docs.microsoft.com/azure/aks/kubernetes-walkthrough)):

```bash
# Declare necessary variables, modify them according to your needs
AKS_RESOURCE_GROUP=akstestrg     # Name of the resource group your AKS cluster will be created in
AKS_NAME=akstest                 # Name of your AKS cluster
AKS_LOCATION=westeurope          # Azure region in which you'll deploy your AKS cluster

# Create the Resource Group where your AKS resource will be installed
az group create --name $AKS_RESOURCE_GROUP --location $AKS_LOCATION

# Create the AKS cluster - this might take some time. Type 'az aks create -h' to see all available options

# The following command will create a four Node AKS cluster. Node size is Standard A1 v1 and Kubernetes version is {{% aks-example-cluster-version %}}. Plus, SSH keys will be generated for you, use --ssh-key-value to provide your values
az aks create --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME --node-count 4 --generate-ssh-keys --node-vm-size Standard_A4_v2 --kubernetes-version {{% aks-example-cluster-version %}} --enable-node-public-ip

# Install kubectl
sudo az aks install-cli

# Get credentials for your new AKS cluster
az aks get-credentials --resource-group $AKS_RESOURCE_GROUP --name $AKS_NAME
```

Alternatively, you can use the [Azure Portal](https://portal.azure.com) to create a new AKS cluster [(instructions)](https://docs.microsoft.com/azure/aks/kubernetes-walkthrough-portal).

### Allowing UDP traffic

For Agones to work correctly, we need to allow UDP traffic to pass through to our AKS cluster. To achieve this, we must update the NSG (Network Security Group) with the proper rule. A simple way to do that is:

* Log in to the [Azure Portal](https://portal.azure.com/)
* Find the resource group where the AKS(Azure Kubernetes Service) resources are kept, which should have a name like `MC_resourceGroupName_AKSName_westeurope`. Alternative, you can type `az resource show --namespace Microsoft.ContainerService --resource-type managedClusters -g $AKS_RESOURCE_GROUP -n $AKS_NAME -o json | jq .properties.nodeResourceGroup`
* Find the Network Security Group object, which should have a name like `aks-agentpool-********-nsg` (ie. aks-agentpool-55978144-nsg for dns-name-prefix agones)
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

### Getting Public IPs to Nodes

#### Kubernetes version prior to 1.18.19, 1.19.11 and 1.20.7

To find a resource's public IP, search for [Virtual Machine Scale Sets](https://portal.azure.com/#blade/HubsExtension/BrowseResourceBlade/resourceType/Microsoft.Compute%2FvirtualMachineScaleSets) -> click on the set name(inside  `MC_resourceGroupName_AKSName_westeurope` group) -> click `Instances` -> click on the instance name -> view `Public IP address`.

To get public IP via API [look here](https://github.com/Azure/azure-libraries-for-net/issues/1185#issuecomment-747919226).


For more information on Public IPs for VM NICs, see [this document](https://docs.microsoft.com/azure/virtual-network/virtual-network-network-interface-addresses). 

#### Kubernetes version starting 1.18.19, 1.19.11 and 1.20.7

Virtual Machines public IP is available directly in Kubernetes EXTERNAL-IP.

## Next Steps

- Continue to [Install Agones]({{< relref "../Install Agones/_index.md" >}}).
