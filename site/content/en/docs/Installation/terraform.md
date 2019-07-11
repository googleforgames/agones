---
title: "Deploy GKE/AKS cluster and install Agones using Terraform"
linkTitle: "Install with Terraform"
weight: 4
description: >
  This chart install the Agones application and defines deployment on a [Kubernetes](http://kubernetes.io) cluster using the Terraform.

---

## Prerequisites

- Terraform v0.11.13
- [Helm](https://docs.helm.sh/helm/) package manager 2.10.0+
- Access to the the Kubernetes hosting provider you are using (e.g. `gcloud` or `az` utility installed)
- Git

# Installing the Agones as Terraform submodule on Google Kubernetes Engine

You can use Terraform to provision your GKE cluster and install agones on it using Helm Terraform provider.

First step would be to enable `Kubernetes Engine API`. From the Cloud Console, navigate to APIs & Services > Dashboard, then click `Enable APIs and Services`. Type `kubernetes` in the search box, and you should find the Kubernetes Engine API. Click Enable.

Install `gcloud` utility by following [these instructions](https://cloud.google.com/sdk/install).

GKE cluster would contain 3 Node Pools:
- Primary Node Pool with `"game-server"` tag, containing 4 nodes.
- `"agones-system"` node pool for Agones Controller.
- `"agones-metrics"` for monitoring and metrics collecting purpose.

Additionally `"tiller"` service account would be created with ClusterRole.

By default you will receive the latest version from [Helm repository](https://agones.dev/chart/stable), but you can configure version using `-var agones-version=<DesiredVersion>`.

## Example and parameters which is configurable

The example of submodule configuration could be found here:
 {{< ghlink href="examples/terraform-submodules/gke/module.tf" >}}Terraform configuration with Agones submodule{{< /ghlink >}}

Configurable parameters and their meaning:
- password - if not specified basic Auth would be disabled in GKE cluster
- agones_version - which version of agones to install
- project - your Google Cloud Project ID
- machine_type - primary cluster machine type ( default is "n1-standard-4")
- node_count - count of nodes in primary Node Pool. Defaults to "4".

## Applying Agones terraform configuration

First you should run:
```
terraform init
```

It would use git to clone the current master of Agones, and use `./build` folder as starting point of Agones submodule, which contains all necessary Terraform configuration files.

Next step you should make sure that you authenticate using gcloud:
```
gcloud auth application-default login
```

Now you are able to deploy properly configured GKE cluster and specify release version of Agones you want to use:
```
terraform apply -var project="<YOUR_GCP_ProjectID>" -var agones_version="0.9.0"
```

Run next command to setup your kubectl:
```
gcloud container clusters get-credentials --zone us-west1-c  test-cluster
```

You would see:
```
Fetching cluster endpoint and auth data.
kubeconfig entry generated for test-cluster.
```

Check that you have access to kubernetes cluster:
```
kubectl get nodes
```

Make sure you have 6 nodes in `Ready` state.

## Uninstall the Agones and delete GKE cluster

Run next command to delete all Terraform provisioned resources:
```
terraform destroy
```

# Installing the Agones as Terraform submodule on Azure Kubernetes Service

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

## Uninstall the Agones and delete AKS cluster

Run next command to delete all Terraform provisioned resources:
```
terraform destroy
```

## Reference 
Details on how you can authenticate your AKS terraform provider using official [instructions](https://www.terraform.io/docs/providers/azurerm/auth/service_principal_client_secret.html)
