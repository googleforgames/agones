// Copyright 2023 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


// Run:
//  terraform apply -var project="<YOUR_GCP_ProjectID>"

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.25.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.3"
    }
  }
}

variable "project" {}
variable "kubernetesVersion" {}

variable "overrideName" {
  default = ""
}

module "gke_cluster" {
  source = "../../../../install/terraform/modules/gke"

  cluster = {
    "name"                 = var.overrideName != "" ? var.overrideName : format("gke-standard-e2e-test-cluster-%s", replace(var.kubernetesVersion, ".", "-"))
    "zone"                 = "us-west1-c"
    "machineType"          = "e2-standard-4"
    "initialNodeCount"     = 10
    "enableImageStreaming" = true
    "project"              = var.project
  }

  udpFirewall = false // firewall is created at the project module level
}

provider "helm" {
  kubernetes {
    host                   = module.gke_cluster.host
    token                  = module.gke_cluster.token
    cluster_ca_certificate = module.gke_cluster.cluster_ca_certificate
  }
}

resource "helm_release" "consul" {
  repository = "https://helm.releases.hashicorp.com"
  chart      = "consul"
  name       = "consul"

  set {
    name  = "server.replicas"
    value = "1"
  }

  set {
    name  = "ui.service.type"
    value = "ClusterIP"
  }

  set {
    name  = "client.enabled"
    value = "false"
  }
}