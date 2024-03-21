#!/usr/bin/env bash

# Copyright 2017 Google LLC All Rights Reserved.
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

#!/bin/bash

set -x
set -o errexit
set -o nounset
set -o pipefail

echo "Starting CRD client generation process..."

echo "Current working directory: $(pwd)"

pushd /go/src/k8s.io/code-generator

echo "Fetching the latest tags from remote..."
git fetch --tags

echo "Checking if the tag v0.30.0-beta.0 exists..."
if git show-ref --tags | grep -q "refs/tags/v0.30.0-beta.0"; then
    echo "Tag found. Checking out v0.30.0-beta.0..."
    git checkout v0.30.0-beta.0
else
    echo "Tag v0.30.0-beta.0 does not exist. Exiting..."
    exit 1
fi

popd


CODEGEN_SCRIPT="/go/src/k8s.io/code-generator/kube_codegen.sh"
echo "Using codegen script at: ${CODEGEN_SCRIPT}"

echo "Sourcing kube_codegen.sh..."
source "${CODEGEN_SCRIPT}"

echo "Generating CRD client code..."
kube::codegen::gen_client \
  /go/src/agones.dev/agones/pkg/apis \
  --output-dir /go/src/agones.dev/agones/pkg/client \
  --boilerplate /go/src/agones.dev/agones/build/boilerplate.go.txt

echo "CRD client code generation complete."

echo "Post-generation working directory: $(pwd)"
