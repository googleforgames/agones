# Copyright 2019 Google LLC All Rights Reserved.
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
  required_version = ">= 0.12.6"
}

data "google_client_config" "default" {}

# A list of all parameters used in interpolation var.cluster
# Set values to default if not key was not set in original map
locals {
  project           = lookup(var.cluster, "project", "agones")
  zone              = lookup(var.cluster, "zone", "us-west1-c")
  name              = lookup(var.cluster, "name", "test-cluster")
  machineType       = lookup(var.cluster, "machineType", "n1-standard-4")
  initialNodeCount  = lookup(var.cluster, "initialNodeCount", "4")
  network           = lookup(var.cluster, "network", "default")
  subnetwork        = lookup(var.cluster, "subnetwork", "")
  kubernetesVersion = lookup(var.cluster, "kubernetesVersion", "1.16")
}

# echo command used for debugging purpose
# Run `terraform taint null_resource.test-setting-variables` before second execution
resource "null_resource" "test-setting-variables" {
  provisioner "local-exec" {
    command = <<EOT
    ${format("echo Current variables set as following - name: %s, project: %s, machineType: %s, initialNodeCount: %s, network: %s, zone: %s",
    local.name, local.project,
    local.machineType, local.initialNodeCount, local.network,
local.zone)}
    EOT
}
}

resource "google_container_cluster" "primary" {
  name       = local.name
  location   = local.zone
  project    = local.project
  network    = local.network
  subnetwork = local.subnetwork

  min_master_version = local.kubernetesVersion

  node_pool {
    name       = "default"
    node_count = local.initialNodeCount
    version = local.kubernetesVersion

    management {
      auto_upgrade = false
    }

    node_config {
      machine_type = local.machineType

      oauth_scopes = [
        "https://www.googleapis.com/auth/devstorage.read_only",
        "https://www.googleapis.com/auth/logging.write",
        "https://www.googleapis.com/auth/monitoring",
        "https://www.googleapis.com/auth/service.management.readonly",
        "https://www.googleapis.com/auth/servicecontrol",
        "https://www.googleapis.com/auth/trace.append",
      ]

      tags = ["game-server"]
    }
  }
  node_pool {
    name       = "agones-system"
    node_count = 1
    version = local.kubernetesVersion

    management {
      auto_upgrade = false
    }

    node_config {
      machine_type = "n1-standard-4"

      oauth_scopes = [
        "https://www.googleapis.com/auth/devstorage.read_only",
        "https://www.googleapis.com/auth/logging.write",
        "https://www.googleapis.com/auth/monitoring",
        "https://www.googleapis.com/auth/service.management.readonly",
        "https://www.googleapis.com/auth/servicecontrol",
        "https://www.googleapis.com/auth/trace.append",
      ]

      labels = {
        "agones.dev/agones-system" = "true"
      }

      taint {
        key    = "agones.dev/agones-system"
        value  = "true"
        effect = "NO_EXECUTE"
      }
    }
  }
  node_pool {
    name       = "agones-metrics"
    node_count = 1
    version = local.kubernetesVersion

    management {
      auto_upgrade = false
    }

    node_config {
      machine_type = "n1-standard-4"

      oauth_scopes = [
        "https://www.googleapis.com/auth/devstorage.read_only",
        "https://www.googleapis.com/auth/logging.write",
        "https://www.googleapis.com/auth/monitoring",
        "https://www.googleapis.com/auth/service.management.readonly",
        "https://www.googleapis.com/auth/servicecontrol",
        "https://www.googleapis.com/auth/trace.append",
      ]

      labels = {
        "agones.dev/agones-metrics" = "true"
      }

      taint {
        key    = "agones.dev/agones-metrics"
        value  = "true"
        effect = "NO_EXECUTE"
      }
    }
  }
  timeouts {
    create = "30m"
    update = "40m"
  }
}

resource "google_compute_firewall" "default" {
  name    = length(var.firewallName) == 0 ? "game-server-firewall-${local.name}" : var.firewallName
  project = local.project
  network = local.network

  allow {
    protocol = "udp"
    ports    = [var.ports]
  }
  target_tags = ["game-server"]
}
