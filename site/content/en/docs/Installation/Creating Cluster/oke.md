---
title: "OCI Kubernetes Engine"
linkTitle: "Oracle Cloud"
weight: 40
description: >
  Follow these steps to create an [OCI Kubernetes Engine (OKE)](https://www.oracle.com/cloud/cloud-native/kubernetes-engine/)
  cluster for your Agones install.
---

Create your OKE Cluster using the [Getting Started Guide](https://docs.oracle.com/en-us/iaas/Content/ContEng/home.htm).

{{% alert title="Note" color="info"%}}
To create a cluster, you must either belong to the tenancy's Administrators group, or belong to a group to which a policy grants the CLUSTER_MANAGE permission. See <a href="https://docs.oracle.com/en-us/iaas/Content/ContEng/Concepts/contengpolicyconfig.htm#Policy_Configuration_for_Cluster_Creation_and_Deployment">Policy Configuration for Cluster Creation and Deployment</a>.
{{% /alert %}}

Possible steps to create OKE cluster through [OCI web console](https://cloud.oracle.com/) are the following:

1. Open the navigation menu and click **Developer Services**. Under **Containers & Artifacts**, click **Kubernetes Clusters (OKE)**.
2. Select the compartment in which you want to create the cluster.
3. On the **Clusters** page, click **Create cluster**.
4. Select one of the following workflows to create the cluster:
   - **Quick Create**: Select this workflow when you only want to specify those properties that are absolutely essential for cluster creation. When you select this option, Kubernetes Engine uses default values for many cluster properties, and creates new network resources as required.
   - **Custom Create**: Select this workflow when you want to be able to specify all of the cluster's properties, use existing network resources, and select advanced options.
5. Click **Submit**.
6. Complete the pages of the workflow you selected. For more information, see:
   - [Using the Console to create a Cluster with Default Settings in the 'Quick Create' workflow](https://docs.oracle.com/en-us/iaas/Content/ContEng/Tasks/contengcreatingclusterusingoke_topic-Using_the_Console_to_create_a_Quick_Cluster_with_Default_Settings.htm#create-quick-cluster)
   - [Using the Console to create a Cluster with Explicitly Defined Settings in the 'Custom Create' workflow](https://docs.oracle.com/en-us/iaas/Content/ContEng/Tasks/contengcreatingclusterusingoke_topic-Using_the_Console_to_create_a_Custom_Cluster_with_Explicitly_Defined_Settings.htm#create-custom-cluster)

## Allowing UDP Traffic

For Agones to work correctly, we need to allow UDP traffic to pass through to our OKE cluster worker nodes. To achieve this, we must update the workers' nodepool SG (Security Group) with the proper rule. A simple way to do that is:

* Log in to the OCI Console
* Go to the Virtual Cloud Network Details Dashboard and select **Network Security Groups**
* Find the Security Group for the workers nodepool, which will be named something like `workers-`
* Select **Add Rules**
* **Edit Rules** to add a new **Custom UDP Rule** with a 7000-8000 port range and an appropriate **Source** CIDR range (`0.0.0.0/0` allows all traffic)

## Next Steps

* Continue to [Install Agones]({{< relref "../Install Agones/_index.md" >}}).
