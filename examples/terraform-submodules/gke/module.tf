// Copyright 2019 Google LLC All Rights Reserved.
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
//  terraform apply -var project="<YOUR_GCP_ProjectID>" [-var agones_version="0.9.0"]
variable "password" {
   default= ""
}

// Install latest version of agones
variable "agones_version" {
  default=""
}
variable "project" {default = "agones"}

variable "machine_type" {default = "n1-standard-4"}

//Additional 2 node pools would be created with 1 node each for "agones-system" and "agones-metrics"
variable "node_count" {default = "4"}
module "agones" {
  source = "git::https://github.com/GoogleCloudPlatform/agones.git//build/?ref=master"
  
  password     = "${var.password}"
  cluster = {
      "zone"             = "us-west1-c"
      "name"             = "test-cluster"
      "machineType"      = "${var.machine_type}"
      "initialNodeCount" = "${var.node_count}"
      "legacyAbac"       = false
      "project"          = "${var.project}"
  }
  agones_version = "${var.agones_version}"
  values_file=""
  chart="agones"
}