// Copyright 2020 Google LLC All Rights Reserved.
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

provider "google" {
  version = "~> 2.10"
}

variable "project" {
  default = ""
}

module "gke_cluster" {
  source = "../../../install/terraform/modules/gke"

  cluster = {
    "name"              = "e2e-test-cluster"
    "zone"              = "us-west1-c"
    "machineType"       = "n1-standard-4"
    "initialNodeCount"  = 8
    "project"           = var.project
  }

  firewallName = "gke-game-server-firewall"
}

provider "helm" {
  version = "~> 1.2"
  kubernetes {
    load_config_file       = false
    host                   = module.gke_cluster.host
    token                  = module.gke_cluster.token
    cluster_ca_certificate = module.gke_cluster.cluster_ca_certificate
  }
}

resource "helm_release" "consul" {
  chart = "stable/consul"
  name = "consul"

  set {
    name = "Replicas"
    value = "1"
  }

  set {
    name = "uiService.type"
    value = "ClusterIP"
  }
}
