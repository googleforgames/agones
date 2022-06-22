# Copyright 2022 Google LLC All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# first we need GKE cluster to host our GameServers
module "gke_cluster" {
  source = "../gke"

  cluster = {
    "name"                    = var.name
    "zone"                    = var.zone
    "machineType"             = var.machine_type
    "initialNodeCount"        = var.node_count
    "project"                 = var.project
    "network"                 = var.network
    "subnetwork"              = var.subnetwork
    "windowsInitialNodeCount" = var.windows_node_count
    "windowsMachineType"      = var.windows_machine_type
    "minNodeCount"            = var.min_node_count
    "maxNodeCount"            = var.max_node_count
  }
  service_account = var.service_account

  providers = {
    gcp = gcp
  }

}

# second we need to install Agones on our fresh GKE cluster
module "helm_agones" {
  source = "../helm3"
  service_account_name = var.service_account_name
  project = var.project
  agones_version                        = var.agones_version
  values_file                           = var.values_file
  feature_gates                         = var.feature_gates
  host                                  = module.gke_cluster.host
  token                                 = module.gke_cluster.token
  cluster_ca_certificate                = module.gke_cluster.cluster_ca_certificate
  log_level                             = var.log_level
  allocationEndpointAgonesPrerequisites = var.enableAllocationEndpoint
}

# this is bit ugly... I know. But it works nontheless :) 
# basically it is what needs to be run on a cluster to prep it for allocation-endpoint from examples folder
# based on https://github.com/googleforgames/agones/tree/main/examples/allocation-endpoint
# it is about running patch-agones-allocator.yaml as described in README.md of allocation-edpoint
# with small additions to handle: 
# 1. Variables instead of fixed project and clusters
# 2. Pulling allocator endpoint ip from cluster and placing it in file (for use in the future) since that is only way to get 
# values from commands started via local-exec and using them as variables in terraform in the future. 
resource "null_resource" "this" {
  depends_on = [module.helm_agones]
  provisioner "local-exec" {
    command = <<-EOT
      gcloud container clusters get-credentials --zone ${var.zone} ${var.name}
      cp ../../allocation-endpoint/patch-agones-allocator.yaml ./modules/gke_with_agones/esp/patch-agones-allocator.yaml
      sed 's/\[GKE-PROJECT-ID\]/${var.project}/g' ./modules/gke_with_agones/esp/patch-agones-allocator.yaml -i
      kubectl patch deployment agones-allocator -n agones-system --patch-file ./modules/gke_with_agones/esp/patch-agones-allocator.yaml
      kubectl patch svc agones-allocator -n agones-system --type merge -p '{"spec":{"ports": [{"port": 443,"name":"https","targetPort":9443}]}}'
      kubectl annotate sa -n agones-system agones-allocator iam.gke.io/gcp-service-account=ae-esp-sa@${var.project}.iam.gserviceaccount.com --overwrite
      kubectl get services --namespace agones-system | grep allocator | grep LoadBalancer | awk '{print $4}' > allocator_ip_${var.name}.txt
      cat allocator_ip_${var.name}.txt | tr --delete '\n\r' > allocator_ip_${var.name}_cleaned.txt
    EOT
  }
  triggers = {
    timestamp = timestamp()
  }
}

# we have allocator ip in file but we need it as an output so we will read the file do data source
data "local_file" "allocator-ip-cluster-1" {
  filename = "./allocator_ip_${var.name}_cleaned.txt"
  depends_on = [null_resource.this]
}

# with the allocator ip in datasource we can finally export it outside the module. 
output "allocator_ip" {
  value = data.local_file.allocator-ip-cluster-1.content
}

output "host" {
  value = module.gke_cluster.host
}

output "token" {
  value     = module.gke_cluster.token
  sensitive = true
}

output "cluster_ca_certificate" {
  value = module.gke_cluster.cluster_ca_certificate
}

output "name" {
  value = var.name
}

output "zone" {
  value = var.zone
}