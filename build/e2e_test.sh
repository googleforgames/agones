#!/usr/bin/env bash

# Copyright 2025 Google LLC All Rights Reserved.
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
set -o pipefail

REGISTRY=$1

export SHELL="/bin/bash"
mkdir -p /go/src/agones.dev/
ln -s /workspace /go/src/agones.dev/agones
cd /go/src/agones.dev/agones/build

pids=()
cloudProducts=("generic" "gke-autopilot")
declare -A versionsAndRegions=( [1.31]=us-east1 [1.32]=us-west1 [1.33]=europe-west1 )

# Keep in sync with the inverse of 'alpha' and 'beta' features in
# pkg/util/runtime/features.go:featureDefaults
featureWithGate="PlayerAllocationFilter=true&FleetAutoscaleRequestMetaData=true&PlayerTracking=true&CountsAndLists=false&RollingUpdateFix=false&PortRanges=false&PortPolicyNone=false&ScheduledAutoscaler=true&AutopilotPassthroughPort=false&GKEAutopilotExtendedDurationPods=false&SidecarContainers=true&Example=true"
featureWithoutGate=""

# Use this if specific feature gates can only be supported on specific Kubernetes versions.
declare -A featureWithGateByVersion=( [1.31]="${featureWithGate}" [1.32]="${featureWithGate}" [1.33]="${featureWithGate}")

for cloudProduct in ${cloudProducts[@]}
do
    for version in "${!versionsAndRegions[@]}"
    do
    withGate=${featureWithGateByVersion[$version]}
    region=${versionsAndRegions[$version]}
    if [ $cloudProduct = generic ]
    then
        testCluster="standard-e2e-test-cluster-${version//./-}"
    else
        testCluster="gke-autopilot-e2e-test-cluster-${version//./-}"
    fi
    testClusterLocation="${region}"
    { stdbuf -oL -eL gcloud builds submit . --suppress-logs --config=./ci/e2e-test-cloudbuild.yaml \
        --substitutions _FEATURE_WITH_GATE=$withGate,_FEATURE_WITHOUT_GATE=$featureWithoutGate,_CLOUD_PRODUCT=$cloudProduct,_TEST_CLUSTER_NAME=$testCluster,_TEST_CLUSTER_LOCATION=$testClusterLocation,_REGISTRY=${REGISTRY},_PARENT_COMMIT_SHA=${COMMIT_SHA},_PARENT_BUILD_ID=${BUILD_ID} \
        |& stdbuf -i0 -oL -eL grep -v " tarball " \
        |& stdbuf -i0 -oL -eL sed "s/^/${cloudProduct}-${version}: /"; } &
    pids+=($!)
    done
done

# If any of the subprocess exit with nonzero code, exit the main process and kill all subprocesses
for pid in "${pids[@]}"; do
    if wait -n; then
    :
    else
    status=$?
    echo "One of the e2e test child cloud build exited with nonzero status $status. Aborting."
    for pid in "${pids[@]}"; do
        # Send a termination signal to all the children, and ignore errors
        # due to children that no longer exist.
        kill "$pid" 2> /dev/null || :
        echo "killed $pid"
    done
    exit "$status"
    fi
done
echo "all done"