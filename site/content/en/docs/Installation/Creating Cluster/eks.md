---
title: "Amazon Elastic Kubernetes Service"
linkTitle: "Amazon Web Services"
weight: 20
description: >
  Follow these steps to create an [Amazon Elastic Kubernetes Service (EKS)](https://aws.amazon.com/eks/)
  cluster for your Agones install.
---

Create your EKS Cluster using the [Getting Started Guide](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html).

Possible steps are the following:

1. Create new IAM role for cluster management.
1. Run `aws configure` to authorize your `awscli` with proper `AWS Access Key ID` and `AWS Secret Access Key`.
1. Create an example cluster:

```bash
eksctl create cluster \
--name prod \
--version {{% eks-example-cluster-version %}} \
--nodegroup-name standard-workers \
--node-type t3.medium \
--nodes 3 \
--nodes-min 3 \
--nodes-max 4
```

{{< alert title="Note" color="info">}}
EKS does not use the normal Kubernetes networking since it
is <a href="https://itnext.io/kubernetes-is-hard-why-eks-makes-it-easier-for-network-and-security-architects-ea6d8b2ca965">incompatible with Amazon VPC networking</a>.
{{< /alert >}}

## Allowing UDP Traffic

For Agones to work correctly, we need to allow UDP traffic to pass through to our EKS cluster worker nodes. To achieve this, we must update the workers' nodepool SG (Security Group) with the proper rule. A simple way to do that is:

* Log in to the AWS Management Console
* Go to the VPC Dashboard and select **Security Groups**
* Find the Security Group for the workers nodepool, which will be named something like `eksctl-[cluster-name]-nodegroup-[cluster-name]-workers/SG`
* Select **Inbound Rules**
* **Edit Rules** to add a new **Custom UDP Rule** with a 7000-8000 port range and an appropriate **Source** CIDR range (`0.0.0.0/0` allows all traffic)

## Use with custom CNI plugins

To ensure compatibility with custom CNI plugins (such as Calico or [Cilium](https://docs.cilium.io/en/stable/installation/k8s-install-helm/)), configure extensions application to use `hostNetwork` for proper functionality via the `agones.extensions.hostNetwork` Agones [Helm]({{< relref "../Install Agones/helm.md" >}}) variable. Also make sure to set the `agones.extensions.http.port` and the `agones.extensions.webhooks.port` values as well.
This requirement arises due to the control plane's inability to reach in-cluster pod endpoints when the data plane operates on a separate network layer ([Github Issue](https://github.com/aws/containers-roadmap/issues/2227)).
Reconfiguring the kube-apiserver is not possible as it is an AWS managed component.

## Next Steps

* Continue to [Install Agones]({{< relref "../Install Agones/_index.md" >}}).
