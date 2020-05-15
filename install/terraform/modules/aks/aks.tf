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

provider "azuread" {
  version = "=0.4.0"
}

provider "azurerm" {
  version = "=2.2.0"

  features {}
}

provider "random" {
  version = "~> 2.2"
}

# Create Service Principal password
resource "azuread_service_principal_password" "aks" {
  end_date             = "2299-12-30T23:00:00Z" # Forever
  service_principal_id = azuread_service_principal.aks.id
  value                = random_string.password.result
}

# Create Azure AD Application for Service Principal
resource "azuread_application" "aks" {
  name = "agones-sp"
}

# Create Service Principal
resource "azuread_service_principal" "aks" {
  application_id = azuread_application.aks.application_id
}

# Generate random string to be used for Service Principal Password
resource "random_string" "password" {
  length  = 32
  special = true
}

resource "azurerm_resource_group" "agones_rg" {
  name     = "agonesRG"
  location = "East US"
}

resource "azurerm_kubernetes_cluster" "agones" {
  name                = var.cluster_name
  location            = azurerm_resource_group.agones_rg.location
  resource_group_name = azurerm_resource_group.agones_rg.name
  dns_prefix          = "agones"

  kubernetes_version = "1.15.10"

  default_node_pool {
    name                = "default"
    node_count          = var.node_count
    vm_size             = var.machine_type
    os_disk_size_gb     = var.disk_size
    enable_auto_scaling = false
    #enable_node_public_ip = true
    #vnet_subnet_id     = azurerm_subnet.aks.id
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
  node_taints           = ["agones.dev/agones-system=true:NoExecute"]
  node_labels           = {
    "agones.dev/agones-system":"true"
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "metrics" {
  name                  = "metrics"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.agones.id
  vm_size               = var.machine_type
  node_count            = 1
  os_disk_size_gb       = var.disk_size
  enable_auto_scaling   = false
  node_taints           = ["agones.dev/agones-metrics=true:NoExecute"]
  node_labels           = {
    "agones.dev/agones-metrics":"true"
  }
}

resource "azurerm_network_security_group" "agones_sg" {
  name                = "agonesSecurityGroup"
  location            = azurerm_resource_group.agones_rg.location
  resource_group_name = azurerm_resource_group.agones_rg.name
}

resource "azurerm_network_security_rule" "gameserver" {
  name                        = "gameserver"
  priority                    = 100
  direction                   = "Inbound"
  access                      = "Allow"
  protocol                    = "UDP"
  source_port_range           = "*"
  destination_port_range      = "7000-8000"
  source_address_prefix       = "*"
  destination_address_prefix  = "*"
  resource_group_name         = azurerm_resource_group.agones_rg.name
  network_security_group_name = azurerm_network_security_group.agones_sg.name
}

resource "azurerm_network_security_rule" "outbound" {
  name                        = "outbound"
  priority                    = 100
  direction                   = "Outbound"
  access                      = "Allow"
  protocol                    = "Tcp"
  source_port_range           = "*"
  destination_port_range      = "*"
  source_address_prefix       = "*"
  destination_address_prefix  = "*"
  resource_group_name         = azurerm_resource_group.agones_rg.name
  network_security_group_name = azurerm_network_security_group.agones_sg.name
}
