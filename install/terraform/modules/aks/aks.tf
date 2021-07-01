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
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "= 2.63.0"
    }
  }
  required_version = ">= 0.12.26"
}

provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "agones_rg" {
  location = var.resource_group_location
  name     = var.resource_group_name
}

resource "azurerm_kubernetes_cluster" "agones" {
  name                = var.cluster_name
  location            = azurerm_resource_group.agones_rg.location
  resource_group_name = azurerm_resource_group.agones_rg.name
  dns_prefix          = "agones"

  kubernetes_version = var.kubernetes_version

  default_node_pool {
    name                  = "default"
    node_count            = var.node_count
    vm_size               = var.machine_type
    os_disk_size_gb       = var.disk_size
    enable_auto_scaling   = false
    enable_node_public_ip = var.enable_node_public_ip
  }

  service_principal {
    client_id     = var.client_id
    client_secret = var.client_secret
  }
  tags = {
    Environment = "Production"
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "system" {
  name                  = "system"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.agones.id
  vm_size               = var.machine_type
  node_count            = 1
  os_disk_size_gb       = var.disk_size
  enable_auto_scaling   = false
  node_taints = [
    "agones.dev/agones-system=true:NoExecute"
  ]
  node_labels = {
    "agones.dev/agones-system" : "true"
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "metrics" {
  name                  = "metrics"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.agones.id
  vm_size               = var.machine_type
  node_count            = 1
  os_disk_size_gb       = var.disk_size
  enable_auto_scaling   = false
  node_taints = [
    "agones.dev/agones-metrics=true:NoExecute"
  ]
  node_labels = {
    "agones.dev/agones-metrics" : "true"
  }
}

data "azurerm_resources" "network_security_groups" {
  resource_group_name = azurerm_kubernetes_cluster.agones.node_resource_group
  type                = "Microsoft.Network/networkSecurityGroups"
}

resource "azurerm_network_security_rule" "gameserver" {
  name                       = "gameserver"
  priority                   = 100
  direction                  = "Inbound"
  access                     = "Allow"
  protocol                   = "Udp"
  source_port_range          = "*"
  destination_port_range     = "7000-8000"
  source_address_prefix      = "*"
  destination_address_prefix = "*"
  # 2021.06.07-WeetA34: Force lowercase to avoid resource recreation due to attribute saved as lowercase
  resource_group_name = lower(data.azurerm_resources.network_security_groups.resource_group_name)
  # Ensure we get the first network security group named aks-agentpool-*******-nsg
  # If an error is raised here, wait few minutes and retry. It seems Azure doesn't always return all resources just after creation.
  network_security_group_name = [for network_security_group in data.azurerm_resources.network_security_groups.resources : network_security_group.name if length(regexall("^aks-agentpool-\\d+-nsg$", network_security_group.name)) > 0][0]
}
