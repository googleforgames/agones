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


variable "name" {
  default = "terratest-cluster"
}

variable "project" {
  default = ""
}

variable "values_file" {
  default = ""
}


module "gke_helm" {
    source = "../../build/terraform/gke/"

    values_file = var.values_file
    project = var.project
    name = var.name
}

// Additional resources for terratests

// Template used in "terraform-google-modules/kubernetes-engine/google//modules/auth"

data "template_file" "kubeconfig" {
  template = file("${path.module}/templates/kubeconfig-template.yaml.tpl")

  vars = {
    context                = var.name
    cluster_ca_certificate = base64encode(module.gke_helm.cluster_ca_certificate)
    endpoint               = module.gke_helm.host
    token                  = module.gke_helm.token
  }
}
resource "local_file" "kubeconfig" {
  content  = data.template_file.kubeconfig.rendered
  filename = "${path.module}/kubeconfig"
}

output "host" {
  value = module.gke_helm.host
}