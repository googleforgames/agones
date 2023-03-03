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
variable "kubernetes_versions_standard" {
  description = "Create standard e2e test clusters with these k8s versions in these zones"
  type        = map(string)
  default     = {
    "1.23" = "us-east1-c"
    "1.24" = "us-west1-c"
    "1.25" = "us-central1-c"
  }
}

variable "kubernetes_versions_autopilot" {
  description = "Create Autopilot e2e test clusters with these k8s versions in these regions"
  type        = map(string)
  default     = {
    "1.23" = "us-east1"
    "1.24" = "us-west1"
    "1.25" = "us-central1"
  }
}

module "gke_standard_cluster" {
  for_each = var.kubernetes_versions_standard
  source = "./gke-standard"
  project = var.project
  kubernetesVersion = each.key
  location = each.value
}

module "gke_autopilot_cluster" {
  for_each = var.kubernetes_versions_autopilot
  source = "./gke-autopilot"
  project = var.project
  kubernetesVersion = each.key
  location = each.value
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