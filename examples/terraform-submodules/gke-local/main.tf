// Copyright 2019 Google LLC All Rights Reserved.
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
//  terraform apply [-var agones_version="0.9.0"]

// Install latest version of agones
variable "agones_version" {
  default = ""
}

// Your GKE project name
variable "project" {
  default = "agones"
}
module "gke_cluster" {

  source = "../../../build/modules/gke"

  cluster = {
    "project"          = "${var.project}"
    "zone"             = "us-west1-c"
    "name"             = "test-cluster3"
    "machineType"      = "n1-standard-4"
    "initialNodeCount" = "4"
  }
}

module "helm_agones" {

  source = "../../../build/modules/helm"

  agones_version         = "${var.agones_version}"
  values_file            = ""
  chart                  = "agones"
  host                   = "${module.gke_cluster.host}"
  token                  = "${module.gke_cluster.token}"
  cluster_ca_certificate = "${module.gke_cluster.cluster_ca_certificate}"
}

output "host" {
  value = "${module.gke_cluster.host}"
}
output "token" {
  value = "${module.gke_cluster.token}"
}
output "cluster_ca_certificate" {
  value = "${module.gke_cluster.cluster_ca_certificate}"
}
