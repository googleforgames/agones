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

provider "google-beta" {
  version = "~> 2.10"
  zone    = "${var.cluster["zone"]}"
}

/*
provider "google" {
  version = "~> 2.10"
}
*/

data "google_client_config" "default" {}

# echo command used for debugging purpose
# Run `terraform taint null_resource.test-setting-variables` before second execution
resource "null_resource" "test-setting-variables" {
  provisioner "local-exec" {
    command = "${"${format("echo Current variables set as following - name: %s, project: %s, machineType: %s, initialNodeCount: %s, zone: %s",
      "${var.cluster["name"]}", "${var.cluster["project"]}",
      "${var.cluster["machineType"]}", "${var.cluster["initialNodeCount"]}",
    "${var.cluster["zone"]}")}"}"
  }
}
resource "google_container_cluster" "primary" {
  name     = "${var.cluster["name"]}"
  location = "${var.cluster["zone"]}"
  project  = "${var.cluster["project"]}"
  provider = "google-beta"

  node_pool {
    name       = "default"
    node_count = "${var.cluster["initialNodeCount"]}"

    node_config {
      machine_type = "${var.cluster["machineType"]}"

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
  name    = "game-server-firewall-firewall-${var.cluster["name"]}"
  project = "${var.cluster["project"]}"
  network = "${google_compute_network.default.name}"

  allow {
    protocol = "udp"
    ports    = ["${var.ports}"]
  }

  source_tags = ["game-server"]
}

resource "google_compute_network" "default" {
  project = "${var.cluster["project"]}"
  name    = "agones-network-${var.cluster["name"]}"
}
