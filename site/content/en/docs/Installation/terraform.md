---
title: "Deploy GKE/AKS cluster and install Agones using Terraform"
linkTitle: "Install with Terraform"
weight: 4
description: >
  Install a [Kubernetes](http://kubernetes.io) cluster and Agones declaratively using Terraform.

---

## Prerequisites

- Terraform v0.12.3
- [Helm](https://docs.helm.sh/helm/) package manager 2.10.0+
- Access to the the Kubernetes hosting provider you are using (e.g. `gcloud` or `az` utility installed)
- Git

## Installing Agones on Google Kubernetes Engine using a Terraform submodule

You can use Terraform to provision a GKE cluster and install Agones on it.

The first step is to enable the `Kubernetes Engine API`. From the Cloud Console, navigate to APIs & Services > Dashboard, then click `Enable APIs and Services`. Type `kubernetes` in the search box to find the Kubernetes Engine API. Click Enable.

Install `gcloud` utility by following [these instructions](https://cloud.google.com/sdk/install).

### Example configuration

An example configuration can be found here:
 {{< ghlink href="examples/terraform-submodules/gke/module.tf" >}}Terraform configuration with Agones submodule{{< /ghlink >}}. Copy the file into a local directory where you will execute the terraform commands.

The GKE cluster created from the example configuration will contain 3 Node Pools:

- `"default"` node pool with `"game-server"` tag, containing 4 nodes.
- `"agones-system"` node pool for Agones Controller.
- `"agones-metrics"` for monitoring and metrics collecting purpose.

Additionally, a `"tiller"` service account will be created with ClusterRole.

Configurable parameters:

- project - your Google Cloud Project ID (required)
- name - the name of the GKE cluster (default is "agones-terraform-example")
- agones_version - the version of agones to install (default is the latest version from the [Helm repository](https://agones.dev/chart/stable))
- machine_type - machine type for hosting game servers (default is "n1-standard-4")
- node_count - count of game server nodes for the default node pool (default is "4")

### Creating the cluster

In the directory where you created `module.tf`, run:
```
terraform init
```

This will cause terraform to clone the Agones repository and use the `./build` folder as starting point of Agones submodule, which contains all necessary Terraform configuration files.

Next make sure that you can authenticate using gcloud:
```
gcloud auth application-default login
```

Now you can create your GKE cluster (optionally specifying the version of Agones you want to use):
```
terraform apply -var project="<YOUR_GCP_ProjectID>" [-var agones_version="1.0.0"]
```

To verify that the cluster was created successfully, set up your kubectl credentials:
```
gcloud container clusters get-credentials --zone us-west1-c agones-terraform-example
```

Then check that you have access to kubernetes cluster:
```
kubectl get nodes
```

You should have 6 nodes in `Ready` state.

To verify that Agones was installed sucessfully, check for any gameservers:
```
kubectl get gameservers
```

You should see none (but no errors).


### Uninstall the Agones and delete GKE cluster

To delete all resources provisioned by Terraform:
```
terraform destroy
```

## Installing the Agones as Terraform submodule on Azure Kubernetes Service

You can deploy Kubernetes cluster on Azure Kubernetes Service and install Agones using terraform.

Install `az` utility by following [these instructions](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest).

The example of AKS submodule configuration could be found here:
 {{< ghlink href="examples/terraform-submodules/aks/module.tf" >}}Terraform configuration with Agones submodule{{< /ghlink >}}

Copy `module.tf` file into a separate folder.

Login to Azure CLI:
```
az login
```

Configure your terraform:
```
terraform init
```

Now you can deploy your cluster (use variables from the above `az ad sp create-for-rbac` command output):
```
terraform apply -var client_id="<appId>" -var client_secret="<password>"
```

Once you created all resources on AKS you can get the credentials so that you can use `kubectl` to configure your cluster:
```
az aks get-credentials --resource-group agonesRG --name test-cluster
```

Check that you have access to kubernetes cluster:
```
kubectl get nodes
```

### Uninstall the Agones and delete AKS cluster

Run next command to delete all Terraform provisioned resources:
```
terraform destroy
```

### Reference
Details on how you can authenticate your AKS terraform provider using official [instructions](https://www.terraform.io/docs/providers/azurerm/auth/service_principal_client_secret.html)
