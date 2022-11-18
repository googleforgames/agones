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
  required_version = ">= 1.0.0"
}

data "google_client_config" "default" {}

# A list of all parameters used in interpolation var.cluster
# Set values to default if not key was not set in original map
locals {
  project                 = lookup(var.cluster, "project", "agones")
  location		  = lookup(var.cluster, "location", "us-west1-c")
  zone                    = lookup(var.cluster, "zone", "")
  name                    = lookup(var.cluster, "name", "test-cluster")
  machineType             = lookup(var.cluster, "machineType", "e2-standard-4")
  initialNodeCount        = lookup(var.cluster, "initialNodeCount", "4")
  enableImageStreaming    = lookup(var.cluster, "enableImageStreaming", true)
  network                 = lookup(var.cluster, "network", "default")
  subnetwork              = lookup(var.cluster, "subnetwork", "")
  kubernetesVersion       = lookup(var.cluster, "kubernetesVersion", "1.23")
  windowsInitialNodeCount = lookup(var.cluster, "windowsInitialNodeCount", "0")
  windowsMachineType      = lookup(var.cluster, "windowsMachineType", "e2-standard-4")
  autoscale 		  = lookup(var.cluster, "autoscale", false)
  minNodeCount		  = lookup(var.cluster, "minNodeCount", "1")
  maxNodeCount		  = lookup(var.cluster, "maxNodeCount", "5")
}

# echo command used for debugging purpose
# Run `terraform taint null_resource.test-setting-variables` before second execution
resource "null_resource" "test-setting-variables" {
  provisioner "local-exec" {
    command = <<EOT
    ${format("echo Current variables set as following - name: %s, project: %s, machineType: %s, initialNodeCount: %s, network: %s, zone: %s, windowsInitialNodeCount: %s, windowsMachineType: %s",
    local.name,
    local.project,
    local.machineType,
    local.initialNodeCount,
    local.network,
    local.zone,
    local.windowsInitialNodeCount,
    local.windowsMachineType,
)}
    EOT
}
}

resource "google_container_cluster" "primary" {
  name       = local.name
  location   = local.zone != "" ? local.zone : local.location
  project    = local.project
  network    = local.network
  subnetwork = local.subnetwork

  min_master_version = local.kubernetesVersion

  node_pool {
    name       = "default"
    node_count = local.autoscale ? null : local.initialNodeCount
    version    = local.kubernetesVersion

    dynamic "autoscaling" {
      for_each = local.autoscale ? [1] : []
      content {
      	min_node_count = local.minNodeCount
	max_node_count = local.maxNodeCount
      }
    }

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

      gcfs_config {
        enabled = local.enableImageStreaming
      }
    }
  }
  node_pool {
    name       = "agones-system"
    node_count = 1
    version    = local.kubernetesVersion

    management {
      auto_upgrade = false
    }

    node_config {
      machine_type = "e2-standard-4"

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

      gcfs_config {
        enabled = true
      }
    }
  }
  node_pool {
    name       = "agones-metrics"
    node_count = 1
    version    = local.kubernetesVersion

    management {
      auto_upgrade = false
    }

    node_config {
      machine_type = "e2-standard-4"

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

      gcfs_config {
        enabled = true
      }
    }
  }
  dynamic "ip_allocation_policy" {
    for_each = tonumber(local.windowsInitialNodeCount) > 0 ? [1] : []
    content {
      # Enable Alias IPs to allow Windows Server networking.
      cluster_ipv4_cidr_block  = "/14"
      services_ipv4_cidr_block = "/20"
    }
  }
  dynamic "node_pool" {
    for_each = tonumber(local.windowsInitialNodeCount) > 0 ? [1] : []
    content {
      name       = "windows"
      node_count = local.windowsInitialNodeCount
      version    = local.kubernetesVersion

      management {
        auto_upgrade = false
      }

      node_config {
        image_type   = "WINDOWS_LTSC"
        machine_type = local.windowsMachineType

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
  source_ranges = [var.sourceRanges]
}
