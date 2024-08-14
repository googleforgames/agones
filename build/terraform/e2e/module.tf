// Copyright 2020 Google LLC All Rights Reserved.
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
//  terraform init -backend-config="bucket=<YOUR_GCP_ProjectID>-e2e-infra-bucket-tfstate" -backend-config="prefix=terraform/state"
//  terraform apply -var project="<YOUR_GCP_ProjectID>"

terraform {
  required_version = ">= 1.0.0"
  required_providers {
    google = {
      source = "hashicorp/google"
      version = "~> 4.25.0"
    }
    helm = {
      source = "hashicorp/helm"
      version = "~> 2.3"
    }
  }
  backend "gcs" {
  }
}

variable "project" {}
variable "kubernetes_versions" {
  description = "Create e2e test clusters with these k8s versions in these regions"
  type        = map(list(string))
  default     = {
    "1.27" = ["us-east1", "RAPID"]
    "1.28" = ["us-west1", "RAPID"]
    "1.29" = ["europe-west1", "RAPID"]
    // "1.30" = ["asia-east1", "RAPID"]
    //
    // Before merge: When adding Kubernetes version 1.{N}, first uncomment the line above, extending
    // the infrastructure to 4 versions temporarily. Come back to these instructions after the
    // update PR merges.
    //
    // After merge: After the Kubernetes update PR merges, and all active PRs are updated:
    //
    // * Move the 1.{N-3} line to the bottom and comment it out
    // * Change the (commented out) 1.{N-3} to 1.{N+1}
    // * You should now have 3 versions uncommented (versions 1.{N-2} .. 1.{N}),
    //   and 1.{N+1} commented out for the next update. The new, commented out 1.{N+1}
    //   should be using the region of the previous 1.{N-3} - this region will become
    //   unused.
    //
    // Rationale: We cycle the regions us-east1 -> us-west1 -> europe-west1 -> asia-east1 -> us-east1
    // as versions are added, using 4 regions so that the PR adding 1.{N} is in a unique region to
    // 1.{N-3} .. 1.{N-1}, meaning versions never need to share a region in CI.
  }
}

module "gke_standard_cluster" {
  for_each = var.kubernetes_versions
  source = "./gke-standard"
  project = var.project
  kubernetesVersion = each.key
  location = each.value[0]
  releaseChannel = each.value[1]
}

module "gke_autopilot_cluster" {
  for_each = var.kubernetes_versions
  source = "./gke-autopilot"
  project = var.project
  kubernetesVersion = each.key
  location = each.value[0]
  releaseChannel = each.value[1]
}

resource "google_compute_firewall" "udp" {
  name    = "gke-game-server-firewall"
  project = var.project
  network = "default"

  allow {
    protocol = "udp"
    ports    = ["7000-8000"]
  }

  target_tags = ["game-server"]
  source_ranges = ["0.0.0.0/0"]
}

resource "google_compute_firewall" "tcp" {
  name    = "gke-game-server-firewall-tcp"
  project = var.project
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["7000-8000"]
  }

  target_tags = ["game-server"]
  source_ranges = ["0.0.0.0/0"]
}
