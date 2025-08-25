#!/usr/bin/env bash

# Copyright 2018 Google LLC All Rights Reserved.
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

FEATURES=$1
CLOUD_PRODUCT=$2
REGISTRY=$3

K8S_MINOR=$(kubectl version -o json | jq -r '.serverVersion.minor' | sed 's/+//')

echo $FEATURES
echo $REGISTRY
echo $K8S_MINOR
set -e

# Ensure we cleanup the finalizers from k8s 1.33 until they patch it
if [ "$K8S_MINOR" = "33" ]; then
    echo "GKE 1.33 detected: cleaning up stuck service finalizers"

    kubectl get svc -n agones-system -o json \
    | jq -r '.items[] | select(.metadata.finalizers | length > 0) | .metadata.name' \
    | awk 'NF' \
    | xargs -r -I {} kubectl patch svc -n agones-system -p '{"metadata":{"finalizers":null}}' --type=merge
fi

echo "installing current release"
DOCKER_RUN= make install FEATURE_GATES='"'$FEATURES'"' REGISTRY='"'$REGISTRY'"' 
echo "starting e2e test"
DOCKER_RUN= make test-e2e ARGS=-parallel=16 E2E_USE_GOTESTSUM=true GOTESTSUM_VERBOSE=true FEATURE_GATES='"'$FEATURES'"' CLOUD_PRODUCT='"'$CLOUD_PRODUCT'"'
echo "completed e2e test"