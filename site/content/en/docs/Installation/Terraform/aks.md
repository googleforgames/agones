---
title: "Installing Agones on Azure Kubernetes Service using Terraform"
linkTitle: "Azure"
weight: 20
description: >
  You can use Terraform to provision an AKS cluster and install Agones on it.
---

## Installation

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

Check that you have access to the Kubernetes cluster:
```
kubectl get nodes
```

Configurable parameters:
- log_level - possible values: Fatal, Error, Warn, Info, Debug (default is "info")

## Uninstall the Agones and delete AKS cluster

Run next command to delete all Terraform provisioned resources:
```
terraform destroy
```

## Reference
Details on how you can authenticate your AKS terraform provider using official [instructions](https://www.terraform.io/docs/providers/azurerm/auth/service_principal_client_secret.html)

## Next Steps

- [Confirm Agones is up and running]({{< relref "../confirm.md" >}})