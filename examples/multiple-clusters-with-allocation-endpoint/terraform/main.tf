// Copyright 2022 Google LLC All Rights Reserved.
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


terraform {
  required_version = ">= 1.0.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.24"
    }
  }
}

# we need dedicated service account for our node pools 
module "service_account" {
  source               = "./modules/service_account"
  project              = var.project
  service_account_name = var.service_account_name
}

# idea was to create it as a seperate entity as possible so we might as well create our own VPC 
resource "google_compute_network" "vpc" {
  name                    = var.network
  auto_create_subnetworks = "false"
  project                 = var.project
}

# naturally if you want more regions / subnets please add more resources 
resource "google_compute_subnetwork" "subnet1" {
  name          = "agones-${var.zone_1}"
  region        = var.region_1
  network       = google_compute_network.vpc.name
  ip_cidr_range = var.cidr_range_1
  project       = var.project
}

resource "google_compute_subnetwork" "subnet2" {
  name          = "agones-${var.zone_2}"
  region        = var.region_2
  network       = google_compute_network.vpc.name
  ip_cidr_range = var.cidr_range_2
  project       = var.project
}

# we have service_account we have subnets let's create clusters ! :) 
module "cluster_1" {
  source               = "./modules/gke_with_agones"
  project              = var.project
  # so yeah for all clusters name will look similar difference will be in region where it was placed. 
  # it will work but if you want more clusters in single region... you need to change it a bit. 
  name                 = "${var.name}-${var.region_1}" 
  agones_version       = var.agones_version
  machine_type         = var.machine_type
  node_count           = var.node_count
  min_node_count       = var.min_node_count
  max_node_count       = var.max_node_count
  zone                 = var.zone_1
  network              = google_compute_network.vpc.name
  subnetwork           = google_compute_subnetwork.subnet1.name
  log_level            = var.log_level
  feature_gates        = var.feature_gates
  windows_node_count   = var.windows_node_count
  windows_machine_type = var.windows_machine_type
  values_file          = var.values_file
  # set enableAllocationEndpoint to true if you want to use Allocation Endpoint module
  # in this example we are assuming you do want it therefore "yes" is a default value
  enableAllocationEndpoint = var.enableAllocationEndpoint
  service_account          = module.service_account.sa_email
  service_account_name     = var.service_account_name

  providers = {
    gcp = google.gcp1
  }
}

module "cluster_2" {
  source                   = "./modules/gke_with_agones"
  project                  = var.project
  name                     = "${var.name}-${var.region_2}"
  agones_version           = var.agones_version
  machine_type             = var.machine_type
  node_count               = var.node_count
  min_node_count           = var.min_node_count
  max_node_count           = var.max_node_count
  zone                     = var.zone_2
  network                  = google_compute_network.vpc.name
  subnetwork               = google_compute_subnetwork.subnet2.name
  log_level                = var.log_level
  feature_gates            = var.feature_gates
  windows_node_count       = var.windows_node_count
  windows_machine_type     = var.windows_machine_type
  values_file              = var.values_file
  enableAllocationEndpoint = var.enableAllocationEndpoint
  service_account          = module.service_account.sa_email
  service_account_name     = var.service_account_name
  providers = {
    gcp = google.gcp1
  }
}

# policy binding for metrics as described in https://agones.dev/site/docs/guides/metrics/#using-stackdriver-with-workload-identity
resource "google_project_iam_binding" "workload_identity" {
  project = var.project
  role    = "roles/iam.workloadIdentityUser"
  members = [
    "serviceAccount:${var.project}.svc.id.goog[agones-system/agones-allocator]",
    "serviceAccount:${var.project}.svc.id.goog[agones-system/agones-controller]",
  ]
}

# we have a cluster, we have policy binding for metrics... lets create allocation-endpoint 
module "allocation_endpoint" {
  source     = "../../allocation-endpoint/terraform"
  project_id = var.project
  authorized_members = [
    "serviceAccount:allocation-endpoint-access@${var.project}.iam.gserviceaccount.com"
  ]
  # clusters_info is a tricky one, if you want to use more than two clustes ypou need to update that variable according to example that 
  # exists already: 
  clusters_info = "[{\"name\":\"cluster_1\",\"endpoint\":\"${module.cluster_1.allocator_ip}\",\"namespace\":\"default\",\"allocation_weight\":50},{\"name\":\"cluster_2\",\"endpoint\":\"${module.cluster_2.allocator_ip}\",\"namespace\":\"default\",\"allocation_weight\":50}]"
  workload-pool = "${var.project}.svc.id.goog"
}

# Purely optional and quite frankly bit of a dirty hack... but we need to create files to pass values from local-exec to variables 
# and otherone we pull from allocator-service (and update with project ID
# The way that was implemented works... but also creates multiple files that are not needed after script finishes with processing 
# this is just a cleanup
resource "null_resource" "cleanup" {
  depends_on = [module.allocation_endpoint]
  provisioner "local-exec" {
    command = <<-EOT
      rm allocator_ip_agones-cluster*.txt
      rm ./modules/gke_with_agones/esp/patch-agones-allocator.yaml
    EOT
  }
  # it will trigger everytime... just in case.
  triggers = {
    timestamp = timestamp()
  }
}

output "assignment_endpoint" {
  value = module.allocation_endpoint.endpoint
}

output "cluster_1_host" {
  value = module.cluster_1.host
}

output "cluster_2_host" {
  value = module.cluster_2.host
}

# optional but I'm lazy so I created that to help me with initial config :) 
output "cluster_1_kubectl_command" {
  value = "gcloud container clusters get-credentials --zone ${module.cluster_1.zone} ${module.cluster_1.name}"
}

output "cluster_2_kubectl_command" {
  value = "gcloud container clusters get-credentials --zone ${module.cluster_2.zone} ${module.cluster_2.name}"
}

# output "cluster_1_cluster_ca_certificate" {
#   value = module.cluster_1.cluster_ca_certificate
# }
# output "cluster_2_cluster_ca_certificate" {
#   value = module.cluster_2.cluster_ca_certificate
# }
