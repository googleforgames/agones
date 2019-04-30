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
  version = "~> 2.4"
  zone      = "${lookup(var.cluster, "zone")}"
}

/*
provider "google" {
  version = "~> 2.4"
}
*/

data "google_client_config" "default" {}

# echo command used for debugging purpose
# Run `terraform taint null_resource.test-setting-variables` before second execution
resource "null_resource" "test-setting-variables" {
    provisioner "local-exec" {
      command = "${"${format("echo Current variables set as following - name: %s, project: %s, machineType: %s, initialNodeCount: %s, zone: %s, legacyAbac: %s",
      "${lookup(var.cluster, "name")}", "${lookup(var.cluster, "project")}",
      "${lookup(var.cluster, "machineType")}", "${lookup(var.cluster, "initialNodeCount")}",
      "${lookup(var.cluster, "zone")}", "${lookup(var.cluster, "legacyAbac")}")}"}"
    }
}


locals {
  username = "${var.password != "" ? var.username : ""}"
}

# assert that password has correct length
# before creating the cluster to avoid 
# unfinished configurations
resource "null_resource" "check-password-length" {
  count = "${length(var.password) >= 16 || length(var.password) == 0 ? 0 : 1}"
  "Password must be more than 16 chars in length" = true
}

resource "google_container_cluster" "primary" {
  name     = "${lookup(var.cluster, "name")}"
  location     = "${lookup(var.cluster, "zone")}"
  project  = "${lookup(var.cluster, "project")}"
  provider = "google-beta"
  # Setting an empty username and password explicitly disables basic auth
  master_auth {
    username = "${local.username}"
    password = "${var.password}"
  }
  remove_default_node_pool = true
  enable_legacy_abac = "${lookup(var.cluster, "legacyAbac")}"
  initial_node_count = "${lookup(var.cluster, "initialNodeCount") + 2}"
}

resource "google_container_node_pool" "agones-gameserver" {
  name       = "default"
  cluster   = "${google_container_cluster.primary.name}"
  location    = "${google_container_cluster.primary.location}"
  project  = "${lookup(var.cluster, "project")}"
  provider = "google-beta"
  node_count = "${lookup(var.cluster, "initialNodeCount")}"
  node_config = {
    machine_type = "${lookup(var.cluster, "machineType")}"
    oauth_scopes = [
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/trace.append",
    ]

    tags = ["game-server"]
    timeouts = {
      create = "30m"
      update = "40m"
    }
  }
}

resource "google_container_node_pool" "agones-system" {
  name       = "agones-system"
  cluster   = "${google_container_cluster.primary.name}"
  location    = "${google_container_cluster.primary.location}"
  project  = "${lookup(var.cluster, "project")}"
  provider = "google-beta"
  node_count = 1
  node_config = {
    preemptible  = true
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
      "stable.agones.dev/agones-system" = "true"
    }
    taint = {
        key = "stable.agones.dev/agones-system"
        value = "true"
        effect = "NO_EXECUTE"
    }
  }
}

resource "google_container_node_pool" "agones-metrics" {
  name       = "agones-metrics"
  cluster   = "${google_container_cluster.primary.name}"
  location    = "${google_container_cluster.primary.location}"
  project  = "${lookup(var.cluster, "project")}"
  provider = "google-beta"
  node_count = 1
  node_config = {
    preemptible  = true
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
      "stable.agones.dev/agones-metrics" = "true"
    }
    taint = {
        key = "stable.agones.dev/agones-metrics"
        value = "true"
        effect = "NO_EXECUTE"
    }
  }
}

resource "google_compute_firewall" "default" {
  name    = "game-server-firewall-firewall-${lookup(var.cluster, "name")}"
  project = "${lookup(var.cluster, "project")}"
  network = "${google_compute_network.default.name}"

  allow {
    protocol = "udp"
    ports    = ["${var.ports}"]
  }

  source_tags = ["game-server"]
}

resource "google_compute_network" "default" {
  project = "${lookup(var.cluster, "project")}"
  name    = "agones-network-${lookup(var.cluster, "name")}"
}