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
//  terraform apply [-var agones_version="1.1.0"]

// Install latest version of agones
variable "agones_version" {
  default=""
}
variable "cluster_name" {
  default="test-cluster"
}

variable "machine_type" {default = "Standard_D2_v2"}

module "aks_cluster" {
  source = "git::https://github.com/googleforgames/agones.git//build/modules/aks/?ref=master"
  
  machine_type = "${var.machine_type}"
  cluster_name = "${var.cluster_name}"
}

module "helm_agones" {
  source = "git::https://github.com/googleforgames/agones.git//build/modules/helm/?ref=master"
  
  agones_version = "${var.agones_version}"
  values_file=""
  chart="agones"
  host="${module.aks_cluster.host}"
  token="${module.aks_cluster.token}"
  cluster_ca_certificate="${module.aks_cluster.cluster_ca_certificate}"
}

output "host" {
    value = "${module.aks_cluster.host}"
}
output "token" {
    value = "${module.aks_cluster.token}"
}
output "cluster_ca_certificate" {
    value = "${module.aks_cluster.cluster_ca_certificate}"
}
