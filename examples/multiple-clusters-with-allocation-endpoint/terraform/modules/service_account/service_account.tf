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

terraform {
  required_version = ">= 1.0.0"
}

resource "google_service_account" "sa" {
  account_id = var.service_account_name
  display_name = var.service_account_name
  project = var.project
}

resource "google_project_iam_member" "role" {
  project = var.project
  role    = "roles/logging.configWriter" 
  member  = "serviceAccount:${google_service_account.sa.email}"
}

output sa_email{
    value = google_service_account.sa.email
}