#!/bin/bash

# Copyright 2018 Google Inc. All Rights Reserved.
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
set -e
export SHELL="/bin/bash"
KIND_PROFILE="agones"
KIND_CONTAINER_NAME="$KIND_PROFILE-control-plane"

if [ -z $(kind get clusters | grep $KIND_PROFILE) ]; then
    echo "Could not find $KIND_PROFILE cluster. Creating...";
    kind create cluster --name $KIND_PROFILE --config /root/kind.yaml --image kindest/node:v1.11.3 --wait 5m;
fi
KIND_KUBE_CONFIG=$(kind get kubeconfig-path --name="$KIND_PROFILE")
docker network connect cloudbuild $KIND_PROFILE-control-plane
KUBECONFIG=$KIND_KUBE_CONFIG kubectl config set-cluster $KIND_PROFILE --server=https://$KIND_CONTAINER_NAME:6443

until KUBECONFIG=$KIND_KUBE_CONFIG kubectl cluster-info; do
        echo "Waiting for cluster to start...";
        sleep 3;
done

while :; do echo 'Go Cyril !'; sleep 1; done

mkdir -p /go/src/agones.dev/agones/
cp -r /workspace/. /go/src/agones.dev/agones/
cd /go/src/agones.dev/agones/build
DOCKER_RUN= make kind-test-cluster
echo "installing current release"