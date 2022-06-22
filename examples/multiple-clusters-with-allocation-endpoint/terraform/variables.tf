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

# your project name
variable "project" {
}

# cluster name... or actually beggining of the name since the other part will be added dinamically
variable "name" {
}

// Install latest version of agones
variable "agones_version" {
}

# Machine_type for worker nodes. It only updates machine types for nodes that will host GameServers. 
variable "machine_type" {
}

// Note: This is the number of gameserver nodes. The Agones module will automatically create an additional
// two node pools with 1 node each for "agones-system" and "agones-metrics".
variable "node_count" {
}

# VPC name
variable "network" {
  description = "The name of the VPC network to attach the cluster and firewall rule to"
}

# subnet name
variable "subnetwork" {
  description = "The subnetwork to host the cluster in. Required field if network value isn't 'default'."
}

# log level 
variable "log_level" {
}

# As described in https://agones.dev/site/docs/guides/feature-stages/
# For example “PlayerTracking=true&ContainerPortAllocation=true”
variable "feature_gates" {
}

# if you need windows nodes 
variable "windows_node_count" {
}

# machine type for windows nodes
variable "windows_machine_type" {
}


variable "values_file" {
}

variable "enableAllocationEndpoint" {
  description = "It needs to be set to true if Allocation Endpoint module is to be used for multi-cluster setup"
}

variable "service_account_name" {
}

# used in autoscaling
variable "min_node_count" {
}

# used as max node number in autoscaling
variable "max_node_count" {
}

# add new variables for region, zone and cidr if you want to use more regions. 
# region for your first cluster
variable "region_1" {
}

# region for the second cluster
variable "region_2" {
}

# zone for the first cluster
variable "zone_1" {
}

variable "zone_2" {
}

# cidr range for first subnet
variable "cidr_range_1" {
}

variable "cidr_range_2" {
}