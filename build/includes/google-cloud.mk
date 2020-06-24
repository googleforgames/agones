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
gcloud-test-cluster: GCP_CLUSTER_NODEPOOL_MACHINETYPE ?= n1-standard-4
gcloud-test-cluster: $(ensure-build-image)
	docker run --rm -it $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) gcloud \
		deployment-manager deployments create $(GCP_CLUSTER_NAME)  \
		--properties cluster.zone:$(GCP_CLUSTER_ZONE),cluster.name:$(GCP_CLUSTER_NAME),cluster.nodePool.initialNodeCount:$(GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT),cluster.nodePool.machineType:$(GCP_CLUSTER_NODEPOOL_MACHINETYPE)\
		--template=$(mount_path)/build/gke-test-cluster/cluster.yml.jinja
	$(MAKE) gcloud-auth-cluster
	$(MAKE) setup-test-cluster

clean-gcloud-test-cluster: $(ensure-build-image)
	docker run --rm -it $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) gcloud \
		deployment-manager deployments delete $(GCP_CLUSTER_NAME)

# Creates a gcloud cluster for end-to-end
# it installs also a consul cluster to handle build system concurrency using a distributed lock
gcloud-e2e-test-cluster: TEST_CLUSTER_NAME ?= e2e-test-cluster
gcloud-e2e-test-cluster: $(ensure-build-image)
	docker run --rm -it $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) gcloud \
		deployment-manager deployments create $(TEST_CLUSTER_NAME) \
		--config=$(mount_path)/build/gke-test-cluster/cluster-e2e.yml
	$(MAKE) gcloud-auth-cluster GCP_CLUSTER_NAME=$(TEST_CLUSTER_NAME) GCP_CLUSTER_ZONE=us-west1-c
	docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) \
		kubectl apply -f $(mount_path)/build/helm.yaml
	docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) \
		helm init --service-account helm --wait
	docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) \
		helm install --wait --set Replicas=1,uiService.type=ClusterIP --name consul stable/consul

# Deletes the gcloud e2e cluster and cleanup any left pvc volumes
clean-gcloud-e2e-test-cluster: TEST_CLUSTER_NAME ?= e2e-test-cluster
clean-gcloud-e2e-test-cluster: $(ensure-build-image)
	-docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) \
		helm delete --purge consul && kubectl delete pvc -l component=consul-consul
	$(MAKE) clean-gcloud-test-cluster GCP_CLUSTER_NAME=$(TEST_CLUSTER_NAME)

# Creates a gcloud cluster for prow
gcloud-prow-build-cluster: $(ensure-build-image)
	docker run --rm -it $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) gcloud \
		deployment-manager deployments create prow-build-cluster \
		--config=$(mount_path)/build/gke-test-cluster/cluster-prow.yml
	$(MAKE) gcloud-auth-cluster GCP_CLUSTER_NAME=prow-build-cluster GCP_CLUSTER_ZONE=us-west1-c

# Deletes the gcloud prow build cluster
clean-gcloud-prow-build-cluster: $(ensure-build-image)
	$(MAKE) clean-gcloud-test-cluster GCP_CLUSTER_NAME=prow-build-cluster

# Pulls down authentication information for kubectl against a cluster, name can be specified through GCP_CLUSTER_NAME
# (defaults to 'test-cluster')
gcloud-auth-cluster: $(ensure-build-image)
	docker run --rm $(common_mounts) $(build_tag) gcloud config set container/cluster $(GCP_CLUSTER_NAME)
	docker run --rm $(common_mounts) $(build_tag) gcloud config set compute/zone $(GCP_CLUSTER_ZONE)
	docker run --rm $(common_mounts) $(build_tag) gcloud container clusters get-credentials $(GCP_CLUSTER_NAME)

# authenticate our docker configuration so that you can do a docker push directly
# to the gcr.io repository
gcloud-auth-docker: $(ensure-build-image)
	docker run --rm $(common_mounts) $(build_tag) gcloud auth print-access-token | docker login -u oauth2accesstoken --password-stdin https://gcr.io

# Clean the gcloud configuration
clean-gcloud-config:
	-sudo rm -r $(build_path)/.config
