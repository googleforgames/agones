# Copyright 2023 Google LLC All Rights Reserved.
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


terraform {
  required_version = ">= 1.0.0"
}

data "google_client_config" "default" {}

# A list of all parameters used in interpolation var.cluster
# Set values to default if not key was not set in original map
locals {
  name              = lookup(var.cluster, "name", "test-cluster")
  project           = lookup(var.cluster, "project", "agones")
  location          = lookup(var.cluster, "location", "us-west1")
  network           = lookup(var.cluster, "network", "default")
  subnetwork        = lookup(var.cluster, "subnetwork", "")
  releaseChannel    = lookup(var.cluster, "releaseChannel", "REGULAR")
  kubernetesVersion = lookup(var.cluster, "kubernetesVersion", "1.24")
}

# echo command used for debugging purpose
# Run `terraform taint null_resource.test-setting-variables` before second execution
resource "null_resource" "test-setting-variables" {
  provisioner "local-exec" {
    command = <<EOT
    ${format("echo Current variables set as following - name: %s, project: %s, location: %s, network: %s, subnetwork: %s, releaseChannel: %s, kubernetesVersion: %s",
    local.name,
    local.project,
    local.location,
    local.network,
    local.subnetwork,
    local.releaseChannel,
    local.kubernetesVersion,
)}
    EOT
}
}

resource "google_container_cluster" "primary" {
  provider = google-beta # required for node_pool_auto_config.network_tags

  name       = local.name
  project    = local.project
  location   = local.location
  network    = local.network
  subnetwork = local.subnetwork

  release_channel {
    channel = local.releaseChannel != "" ? local.releaseChannel : "UNSPECIFIED"
  }
  min_master_version = local.kubernetesVersion

  enable_autopilot = true
  ip_allocation_policy {} # https://github.com/hashicorp/terraform-provider-google/issues/10782#issuecomment-1024488630

  node_pool_auto_config {
    network_tags {
      tags = ["game-server"]
    }
  }

  timeouts {
    create = "40m"
    update = "60m"
  }
}

resource "google_compute_firewall" "default" {
  count   = var.udpFirewall ? 1 : 0
  name    = length(var.firewallName) == 0 ? "game-server-firewall-${local.name}" : var.firewallName
  project = local.project
  network = local.network

  allow {
    protocol = "udp"
    ports    = [var.ports]
  }

  target_tags   = ["game-server"]
  source_ranges = [var.sourceRanges]
}
