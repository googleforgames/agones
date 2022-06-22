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

#
# This is a Helm 3.x module, and is the module that should be actively used.
#
terraform {
  required_version = ">= 1.0.0"
  required_providers {
    helm = {
      version = "~> 2.3"
      source  = "hashicorp/helm"
    }
  }
}

resource "helm_release" "agones" {
  name             = "agones"
  repository       = "https://agones.dev/chart/stable"
  force_update     = var.force_update
  chart            = var.chart
  timeout          = 420
  version          = var.agones_version
  namespace        = "agones-system"
  create_namespace = true

  # Use terraform of the latest >=1.0.0 version
  values = [
    length(var.values_file) == 0 ? "" : file(var.values_file),
  ]

  set {
    name  = "crds.CleanupOnDelete"
    value = var.crd_cleanup
  }

  set {
    name  = "agones.image.registry"
    value = var.image_registry
  }

  set {
    name  = "agones.image.controller.pullPolicy"
    value = var.pull_policy
  }

  set {
    name  = "agones.image.sdk.alwaysPull"
    value = var.always_pull_sidecar
  }

  set {
    name  = "agones.image.controller.pullSecret"
    value = var.image_pull_secret
  }

  set {
    name  = "agones.ping.http.serviceType"
    value = var.ping_service_type
  }

  set {
    name  = "agones.ping.udp.expose"
    value = var.udp_expose
  }

  set {
    name  = "agones.ping.udp.serviceType"
    value = var.ping_service_type
  }

  set {
    name  = "agones.controller.logLevel"
    value = var.log_level
  }

  set {
    name  = "agones.featureGates"
    value = var.feature_gates
  }

  set {
    name  = "gameservers.namespaces"
    value = "{${join(",", var.gameserver_namespaces)}}"
  }

  set {
    name  = "gameservers.minPort"
    value = var.gameserver_minPort
  }

  set {
    name  = "gameservers.maxPort"
    value = var.gameserver_maxPort
  }
  # allocation service prereq
  set {
    name  = "agones.allocator.disableMTLS"
    value = var.allocationEndpointAgonesPrerequisites ? "true" : "false"
  }
  set {
    name  = "agones.allocator.disableTLS"
    value = var.allocationEndpointAgonesPrerequisites ? "true" : "false"
  }
  set {
    name  = "agones.allocator.service.http.enabled"
    value = var.allocationEndpointAgonesPrerequisites ? "false" : "true"
  }
  # /allocation service prereq
  # enable stackdriver integration 
  set {
    name  = "agones.metrics.stackdriverEnabled"
    value = "true"
  }  
  set {
    name  = "agones.metrics.prometheusEnabled"
    value = "false"
  }  
  set {
    name  = "agones.metrics.prometheusServiceDiscovery"
    value = "false"
  }
  # annotations that needs to be added to handle monitoring with workload identity 
  set {
    name = "agones.serviceaccount.allocator.annotations.iam\\.gke\\.io/gcp-service-account"
    value = "${var.service_account_name}@${var.project}.iam.gserviceaccount.com"
  }

  set {
    name = "agones.serviceaccount.controller.annotations.iam\\.gke\\.io/gcp-service-account"
    value = "${var.service_account_name}@${var.project}.iam.gserviceaccount.com"
  }
}

