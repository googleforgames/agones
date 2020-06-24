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
//  terraform apply [-var agones_version="1.3.0"]

// Install latest version of agones
variable "agones_version" {
  default = ""
}

variable "cluster_name" {
  default = "test-cluster"
}

variable "node_count" {
  default = 4
}

variable "disk_size" {
  default = 30
}

variable "client_id" {
  default = ""
}
variable "client_secret" {
  default = ""
}

variable "machine_type" { default = "Standard_D2_v2" }

variable "log_level" {
  default = "info"
}

variable "feature_gates" {
  default = ""
}

module "aks_cluster" {
  source = "git::https://github.com/googleforgames/agones.git//install/terraform/modules/aks/?ref=master"

  machine_type  = var.machine_type
  cluster_name  = var.cluster_name
  node_count    = var.node_count
  disk_size     = var.disk_size
  client_id     = var.client_id
  client_secret = var.client_secret
}

module "helm_agones" {
  source = "git::https://github.com/googleforgames/agones.git//install/terraform/modules/helm3/?ref=master"

  agones_version         = var.agones_version
  values_file            = ""
  feature_gates          = var.feature_gates
  host                   = module.aks_cluster.host
  token                  = module.aks_cluster.token
  cluster_ca_certificate = module.aks_cluster.cluster_ca_certificate
  log_level              = var.log_level
}

output "host" {
  value = module.aks_cluster.host
}
output "token" {
  value = module.aks_cluster.token
}
output "cluster_ca_certificate" {
  value = module.aks_cluster.cluster_ca_certificate
}
