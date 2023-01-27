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
set -e
FEATURES=$1
CLOUD_PRODUCT=$2
TEST_CLUSTER_NAME=$3
TEST_CLUSTER_LOCATION=$4
REGISTRY=$5

echo $FEATURES
export SHELL="/bin/bash"
export KUBECONFIG="/root/.kube/config"
mkdir -p /go/src/agones.dev/ /root/.kube/
ln -s /workspace /go/src/agones.dev/agones
cd /go/src/agones.dev/agones/build
if [ "$1" = 'local' ]
then
        gcloud auth login
fi
gcloud container clusters get-credentials $TEST_CLUSTER_NAME \
        --zone=${TEST_CLUSTER_LOCATION} --project=agones-images

# TODO: Here we're using the presence of consul to dictate whether we use consul
# port forwarding or whether we rely on Cloud Build serialization from #2932. This
# allows us to quickly recover (by reinstalling consul) if something breaks.
# After a few more days of PRs, it should be safe to remove this.
if kubectl get statefulset/consul-consul-server -oname >& /dev/null
then
        echo "Using legacy consul locking"
        kubectl port-forward statefulset/consul-consul-server 8500:8500 &
        echo "Waiting consul port-forward to launch on 8500..."
        timeout 60 bash -c 'until printf "" 2>>/dev/null >>/dev/tcp/$0/$1; do sleep 1; done' 127.0.0.1 8500
        echo "consul port-forward launched. Starting e2e tests..."
        echo "consul lock -child-exit-code=true -timeout 90m -verbose LockE2E '/root/e2e.sh "$FEATURES" "$CLOUD_PRODUCT" "$REGISTRY"'"
        consul lock -child-exit-code=true -timeout 90m -verbose LockE2E '/root/e2e.sh "'$FEATURES'" "'$CLOUD_PRODUCT'" "'$REGISTRY'"'
        killall -q kubectl || true
        echo "successfully killed kubectl proxy"
else
        echo /root/e2e.sh "${FEATURES}" "${CLOUD_PRODUCT}" "${REGISTRY}"
        /root/e2e.sh "${FEATURES}" "${CLOUD_PRODUCT}" "${REGISTRY}"
fi
