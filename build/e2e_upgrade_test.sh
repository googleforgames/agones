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

BASE_VERSION=$1
PROJECT_ID=$2
logBucketName="upgrade-test-container-logs"

apt-get update && apt-get install -y jq
export SHELL="/bin/bash"
export KUBECONFIG="/root/.kube/config"
mkdir -p /go/src/agones.dev/ /root/.kube/
ln -s /workspace /go/src/agones.dev/agones
cd /go/src/agones.dev/agones/test/upgrade

# --- Function to print failure logs ---
print_failure_logs() {
    local cluster_name=$1
    echo "ERROR: Upgrade test failed on cluster: $cluster_name"

    job_pods=$(kubectl get pods -l job-name=upgrade-test-runner -o jsonpath="{.items[*].metadata.name}")
    kubectl logs --tail=20 "$job_pods" || echo "Unable to retrieve logs for pod: $job_pods"
    for pod in $job_pods; do
        containers=$(kubectl get pod "$pod" -o jsonpath='{.spec.containers[*].name}')
        for container in $containers; do
            if [[ "$container" == "sdk-client-test" || "$container" == "upgrade-test-controller" ]]; then
                echo "----- Logs from pod: $pod, container: $container -----"
                kubectl logs "$pod" -c "$container" || echo "Failed to retrieve logs from $pod/$container"
            fi
        done
    done

    log_url="https://console.cloud.google.com/logs/query;storageScope=storage,projects%2F${PROJECT_ID}%2Flocations%2Fglobal%2Fbuckets%2Fupgrade-test-container-logs%2Fviews%2F_AllLogs;query=$(printf 'resource.labels.container_name=(\"upgrade-test-controller\" OR \"sdk-client-test\")' | jq -sRr @uri);cursorTimestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ);duration=PT10M?project=${PROJECT_ID}"
    echo "Logs from log bucket: $log_url"
}
# ------------------------------------------------------

pids=()
typeset -A waitPids    # Associative array for mapping `kubectl wait job` pid -> `kubectl wait job` output log name
tmpdir=$(mktemp -d)
trap 'rm -rf -- "$tmpdir"' EXIT SIGTERM

# Update image tags to include the current build version.
DevVersion="${BASE_VERSION}-dev-$(git rev-parse --short=7 HEAD)"
export DevVersion
sed "s/\${DevVersion}/${DevVersion}/" upgradeTest.yaml > "${tmpdir}"/upgradeTest.yaml
sed "s/\${DevVersion}/${DevVersion}/" versionMap.yaml > "${tmpdir}"/versionMap.yaml

# Kill all currently running child processes on exit or if a non-zero signal is seen
trap 'echo Cleaning up any remaining running pids: $(jobs -p) ; kill $(jobs -p) 2> /dev/null || :' EXIT SIGTERM

cloudProducts=("generic" "gke-autopilot")
declare -A versionsAndRegions=( [1.33]=us-central1 [1.32]=us-west1 [1.31]=us-east1 )

for cloudProduct in "${cloudProducts[@]}"
do
    for version in "${!versionsAndRegions[@]}"
    do
    region=${versionsAndRegions[$version]}
    if [ "$cloudProduct" = generic ]; then
        testCluster="standard-upgrade-test-cluster-${version//./-}"
    else
        testCluster="gke-autopilot-upgrade-test-cluster-${version//./-}"
    fi
    testClusterLocation="${region}"

    echo "===== Processing cluster: $testCluster in $testClusterLocation ====="

    gcloud container clusters get-credentials "$testCluster" --region="$testClusterLocation" --project="$PROJECT_ID"

    if [ "$cloudProduct" = gke-autopilot ]; then
        # For autopilot clusters use evictable "balloon" pods to keep a buffer in node pool autoscaling.
        kubectl apply -f evictablePods.yaml
    fi

    # Clean up any existing job / namespace / apiservice from previous run
    echo Checking if resources from a previous build of upgrade-test-runner exist and need to be cleaned up on cluster "${testCluster}".
    if kubectl get jobs | grep upgrade-test-runner ; then
        echo Deleting job from previous run of upgrade-test-runner on cluster "${testCluster}".
        kubectl delete job upgrade-test-runner
        kubectl wait --for=delete pod -l job-name=upgrade-test-runner --timeout=5m
    fi

    # Check if there are any dangling game servers.
    if kubectl get gs | grep ".*"; then
        # Remove any finalizers so that dangling game servers can be manually deleted.
        kubectl get gs -o=custom-columns=:.metadata.name --no-headers | xargs kubectl patch gs -p '{"metadata":{"finalizers":[]}}' --type=merge
        sleep 5
        echo Deleting game servers from previous run of upgrade-test-runner on cluster "${testCluster}".
        kubectl delete gs -l app=sdk-client-test
    fi

    if kubectl get po -l app=sdk-client-test | grep ".*"; then
        echo Deleting pods from previous run of upgrade-test-runner on cluster "${testCluster}".
        kubectl delete po -l app=sdk-client-test
        kubectl wait --for=delete pod -l app=sdk-client-test --timeout=5m
    fi

    # The v1.allocation.agones.dev apiservice does not get removed automatically and will prevent the namespace from terminating.
    if kubectl get apiservice | grep v1.allocation.agones.dev ; then
        echo Deleting v1.allocation.agones.dev from previous run of upgrade-test-runner on cluster "${testCluster}".
        kubectl delete apiservice v1.allocation.agones.dev
    fi

    if kubectl get namespace | grep agones-system ; then
        echo Deleting agones-system namespace from previous run of upgrade-test-runner on cluster "${testCluster}".
        kubectl delete namespace agones-system
        kubectl wait --for=delete ns agones-system --timeout=5m
    fi

    if kubectl get crds | grep agones ; then
        echo Deleting crds from previous run of upgrade-test-runner on cluster "${testCluster}".
        kubectl get crds -o=custom-columns=:.metadata.name | grep agones | xargs kubectl delete crd
    fi

    echo kubectl apply -f permissions.yaml on cluster "${testCluster}"
    kubectl apply -f permissions.yaml
    echo kubectl apply -f versionMap.yaml on cluster "${testCluster}"
    kubectl apply -f "${tmpdir}"/versionMap.yaml
    echo kubectl apply -f gameserverTemplate.yaml on cluster "${testCluster}"
    kubectl apply -f gameserverTemplate.yaml

    echo kubectl apply -f upgradeTest.yaml on cluster "${testCluster}"
    kubectl apply -f "${tmpdir}"/upgradeTest.yaml

    # We need to wait for job pod to be created and ready before we can wait on the job itself.
    # TODO: Once all test clusters are at Kubernetes Version >= 1.31 use `kubectl wait --for=create` instead of sleep.
    # kubectl wait --for=create pod -l job-name=upgrade-test-runner --timeout=1m
    sleep 10s

    # Wait for the pod to become ready (or timeout)
    if ! kubectl wait --for=condition=ready pod -l job-name=upgrade-test-runner --timeout=5m; then
        echo "ERROR: The pod for job 'upgrade-test-runner' did not become ready within the timeout period."
        print_failure_logs "$testCluster"
        exit 1
    fi

    echo Wait for job upgrade-test-runner to complete or fail on cluster "${testCluster}"
    kubectl wait job/upgrade-test-runner --timeout=30m --for jsonpath='{.status.conditions[*].status}'=True -o jsonpath='{.status.conditions[*]}' | tee "${tmpdir}"/"${testCluster}".log &
    waitPid=$!
    pids+=( "$waitPid" )
    waitPids[$waitPid]="${tmpdir}"/"${testCluster}".log

    done
done

for pid in "${pids[@]}"; do
    # This block executes when the process exits and pid status==0
    if wait $pid; then
        outputLog="${waitPids[$pid]}"
        # wait for output to finish writing to file
        until [ -s "$outputLog" ]; do sleep 1; done
        output_json=$(<"${outputLog}")

        echo "Reading output from log file: $outputLog:"
        echo "$output_json" | jq '.'

        job_condition_type=$(echo "$output_json" | jq -r '.type')
        job_condition_message=$(echo "$output_json" | jq -r '.message')

        # "Complete" is successful job run.
        # Version 1.31 has "SuccessCriteriaMet" as the first completion status returned, or "FailureTarget" in case of failure.
        if [ "$job_condition_type" == "Complete" ] || [ "$job_condition_type" == "SuccessCriteriaMet" ]; then
            echo "Job completed successfully on cluster associated with log: $outputLog"
            continue
        else
            echo "Unexpected job status: '$job_condition_type' with message: '$job_condition_message' in log $outputLog"
            print_failure_logs "$(basename "$outputLog" .log)"
            exit 1
        fi
    # This block executes when the process exits and pid status!=0
    else
        status=$?
        outputLog="${waitPids[$pid]}"
        echo "One of the upgrade tests pid $pid from cluster log $outputLog exited with a non-zero status ${status}."
        print_failure_logs "$(basename "$outputLog" .log)"
        exit $status
    fi
done

echo "End of Upgrade Tests"
