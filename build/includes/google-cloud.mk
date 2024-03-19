# Copyright 2019 Google LLC All Rights Reserved.
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

#    ____                   _         ____ _                 _
#   / ___| ___   ___   __ _| | ___   / ___| | ___  _   _  __| |
#  | |  _ / _ \ / _ \ / _` | |/ _ \ | |   | |/ _ \| | | |/ _` |
#  | |_| | (_) | (_) | (_| | |  __/ | |___| | (_) | |_| | (_| |
#   \____|\___/ \___/ \__, |_|\___|  \____|_|\___/ \__,_|\__,_|
#                     |___/

# Initialise the gcloud login and project configuration, if you are working with GCP
gcloud-init: ensure-build-config
	docker run --rm -it $(common_mounts) $(build_tag) gcloud init

# Creates and authenticates a small, 6 node GKE cluster to work against (2 nodes are used for agones-metrics and agones-system)
gcloud-test-cluster: GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT ?= 4
gcloud-test-cluster: GCP_CLUSTER_NODEPOOL_MACHINETYPE ?= e2-standard-4
gcloud-test-cluster: GCP_CLUSTER_NODEPOOL_ENABLEIMAGESTREAMING ?= true
gcloud-test-cluster: GCP_CLUSTER_NODEPOOL_AUTOSCALE ?= false
gcloud-test-cluster: GCP_CLUSTER_NODEPOOL_MIN_NODECOUNT ?= 1
gcloud-test-cluster: GCP_CLUSTER_NODEPOOL_MAX_NODECOUNT ?= 5
gcloud-test-cluster: GCP_CLUSTER_NODEPOOL_WINDOWSINITIALNODECOUNT ?= 0
gcloud-test-cluster: GCP_CLUSTER_NODEPOOL_WINDOWSMACHINETYPE ?= e2-standard-4
gcloud-test-cluster: GCP_CLUSTER_NODEPOOL_ENABLE_AGONES_METRICS_NODEPOOL ?= true
gcloud-test-cluster: $(ensure-build-image)
	$(MAKE) gcloud-terraform-cluster GCP_TF_CLUSTER_NAME="$(GCP_CLUSTER_NAME)" \
		GCP_CLUSTER_LOCATION="$(GCP_CLUSTER_LOCATION)" \
		GCP_CLUSTER_AUTOSCALE="$(GCP_CLUSTER_NODEPOOL_AUTOSCALE)" \
		GCP_CLUSTER_MIN_NODECOUNT="$(GCP_CLUSTER_NODEPOOL_MIN_NODECOUNT)" \
		GCP_CLUSTER_MAX_NODECOUNT="$(GCP_CLUSTER_NODEPOOL_MAX_NODECOUNT)" \
		GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT="$(GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT)" \
		GCP_CLUSTER_NODEPOOL_MACHINETYPE="$(GCP_CLUSTER_NODEPOOL_MACHINETYPE)" \
		GCP_CLUSTER_NODEPOOL_ENABLEIMAGESTREAMING="$(GCP_CLUSTER_NODEPOOL_ENABLEIMAGESTREAMING)" \
		GCP_CLUSTER_NODEPOOL_WINDOWSINITIALNODECOUNT="$(GCP_CLUSTER_NODEPOOL_WINDOWSINITIALNODECOUNT)" \
		GCP_CLUSTER_NODEPOOL_WINDOWSMACHINETYPE="$(GCP_CLUSTER_NODEPOOL_WINDOWSMACHINETYPE)" \
		GCP_CLUSTER_NODEPOOL_ENABLE_AGONES_METRICS_NODEPOOL="$(GCP_CLUSTER_NODEPOOL_ENABLE_AGONES_METRICS_NODEPOOL)"
	$(MAKE) gcloud-auth-cluster

clean-gcloud-test-cluster: $(ensure-build-image)
	$(MAKE) gcloud-terraform-destroy-cluster

gcloud-e2e-infra-state-bucket: GCP_PROJECT ?= $(shell $(current_project))
gcloud-e2e-infra-state-bucket:
	$(MAKE) terraform-init DIRECTORY=e2e/state-bucket
	docker run --rm -it $(common_mounts) $(build_tag) bash -c 'cd $(mount_path)/build/terraform/e2e/state-bucket && \
      	terraform apply -auto-approve -var project="$(GCP_PROJECT)"'

ensure-e2e-infra-state-bucket: GCP_PROJECT ?= $(shell $(current_project))
ensure-e2e-infra-state-bucket:
	@buckets=$$(docker run --rm $(common_mounts) $(build_tag) gcloud storage buckets describe gs://$(GCP_PROJECT)-e2e-infra-bucket-tfstate --format="value(name)");\
	if [ -z $${buckets} ]; then\
		echo "** E2E infra state bucket $(GCP_PROJECT)-e2e-infra-bucket-tfstate does not exist (or gcloud failed), creating...";\
		$(MAKE) gcloud-e2e-infra-state-bucket;\
	fi

# Creates a gcloud cluster for end-to-end
gcloud-e2e-test-cluster: GCP_PROJECT ?= $(shell $(current_project))
gcloud-e2e-test-cluster: $(ensure-build-image) ensure-e2e-infra-state-bucket
gcloud-e2e-test-cluster:
	$(MAKE) terraform-init BUCKET=$(GCP_PROJECT)-e2e-infra-bucket-tfstate PREFIX=terraform/state DIRECTORY=e2e
	docker run --rm -it $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) bash -c 'cd $(mount_path)/build/terraform/e2e && \
      	terraform apply -auto-approve -var project="$(GCP_PROJECT)"'

# Deletes the gcloud e2e cluster and cleanup any left pvc volumes
clean-gcloud-e2e-test-cluster: GCP_PROJECT ?= $(shell $(current_project))
clean-gcloud-e2e-test-cluster: $(ensure-build-image)
clean-gcloud-e2e-test-cluster:
	$(MAKE) terraform-init BUCKET=$(GCP_PROJECT)-e2e-infra-bucket-tfstate PREFIX=terraform/state DIRECTORY=e2e
	$(DOCKER_RUN) bash -c 'cd $(mount_path)/build/terraform/e2e && terraform destroy -var project=$(GCP_PROJECT) -auto-approve'

# Pulls down authentication information for kubectl against a cluster, name can be specified through GCP_CLUSTER_NAME
# (defaults to 'test-cluster')
gcloud-auth-cluster: $(ensure-build-image)
	docker run --rm $(common_mounts) $(build_tag) gcloud container clusters get-credentials $(GCP_CLUSTER_NAME) --zone  $(GCP_CLUSTER_LOCATION)

# authenticate our docker configuration so that you can do a docker push directly
# to the Google Artifact Registry repository
gcloud-auth-docker: $(ensure-build-image)
	docker run --rm $(common_mounts) $(build_tag) gcloud auth print-access-token | docker login -u oauth2accesstoken --password-stdin https://us-docker.pkg.dev

# Clean the gcloud configuration
clean-gcloud-config:
	-sudo rm -r $(build_path)/.config
