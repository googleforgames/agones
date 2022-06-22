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


variable "project" {
}
variable "name" {
}

variable "agones_version" {
}

variable "machine_type" {}

// Note: This is the number of gameserver nodes. The Agones module will automatically create an additional
// two node pools with 1 node each for "agones-system" and "agones-metrics".
variable "node_count" {
}

variable "zone" {
}

variable "network" {
}

variable "subnetwork" {
}

variable "log_level" {
}

variable "feature_gates" {
}

variable "windows_node_count" {
}

variable "windows_machine_type" {
}

variable "values_file"{
  default = ""
}

variable "enableAllocationEndpoint" {
}

variable "service_account" {
} 

variable "service_account_name" {
}

variable "min_node_count" {
}

variable "max_node_count" {
}