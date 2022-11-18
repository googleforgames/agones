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

# The GKE development cluster name
GCP_TF_CLUSTER_NAME ?= agones-tf-cluster

# the current project
current_project := $(shell $(DOCKER_RUN) bash -c "gcloud config get-value project 2> /dev/null")

### Deploy cluster with Terraform
terraform-init: TERRAFORM_BUILD_DIR ?= $(mount_path)/build/terraform/$(DIRECTORY)
terraform-init: $(ensure-build-image)
terraform-init:
	docker run --rm -it $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) bash -c '\
	cd $(TERRAFORM_BUILD_DIR) && terraform init && gcloud auth application-default login'

terraform-clean: TERRAFORM_BUILD_DIR ?= $(mount_path)/build/terraform/gke
terraform-clean:
	$(DOCKER_RUN) bash -c ' \
	cd $(TERRAFORM_BUILD_DIR) && rm -r ./.terraform || true && \
	rm ./terraform.tfstate* || true && \
	rm ./kubeconfig || true'

# Creates a cluster and install release version of Agones controller
# Version could be specified by AGONES_VERSION
# Alpha Feature gates are disabled
gcloud-terraform-cluster: GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT ?= 4
gcloud-terraform-cluster: GCP_CLUSTER_NODEPOOL_MACHINETYPE ?= e2-standard-4
gcloud-terraform-cluster: GCP_CLUSTER_NODEPOOL_ENABLEIMAGESTREAMING ?= true
gcloud-terraform-cluster: GCP_CLUSTER_NODEPOOL_AUTOSCALE ?= false
gcloud-terraform-cluster: GCP_CLUSTER_NODEPOOL_MIN_NODECOUNT ?= 1
gcloud-terraform-cluster: GCP_CLUSTER_NODEPOOL_MAX_NODECOUNT ?= 5
gcloud-terraform-cluster: GCP_CLUSTER_NODEPOOL_WINDOWSINITIALNODECOUNT ?= 0
gcloud-terraform-cluster: GCP_CLUSTER_NODEPOOL_WINDOWSMACHINETYPE ?= e2-standard-4
gcloud-terraform-cluster: AGONES_VERSION ?= ''
gcloud-terraform-cluster: GCP_TF_CLUSTER_NAME ?= agones-tf-cluster
gcloud-terraform-cluster: LOG_LEVEL ?= debug
gcloud-terraform-cluster: $(ensure-build-image)
gcloud-terraform-cluster: FEATURE_GATES := ""
gcloud-terraform-cluster: GCP_PROJECT ?= $(current_project)
gcloud-terraform-cluster:
	$(MAKE) terraform-init DIRECTORY=gke
	$(DOCKER_RUN) bash -c 'cd $(mount_path)/build/terraform/gke && \
		 terraform apply -auto-approve -var agones_version="$(AGONES_VERSION)" \
		-var name=$(GCP_TF_CLUSTER_NAME) -var machine_type="$(GCP_CLUSTER_NODEPOOL_MACHINETYPE)" \
		-var values_file="" \
		-var feature_gates=$(FEATURE_GATES) \
		-var project="$(GCP_PROJECT)" \
		-var location="$(GCP_CLUSTER_LOCATION)" \
		-var log_level="$(LOG_LEVEL)" \
		-var autoscale=$(GCP_CLUSTER_NODEPOOL_AUTOSCALE) \
		-var min_node_count=$(GCP_CLUSTER_NODEPOOL_MIN_NODECOUNT) \
		-var max_node_count=$(GCP_CLUSTER_NODEPOOL_MAX_NODECOUNT) \
		-var node_count=$(GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT) \
		-var enable_image_streaming=$(GCP_CLUSTER_NODEPOOL_ENABLEIMAGESTREAMING) \
		-var windows_node_count=$(GCP_CLUSTER_NODEPOOL_WINDOWSINITIALNODECOUNT) \
		-var windows_machine_type=$(GCP_CLUSTER_NODEPOOL_WINDOWSMACHINETYPE)'
	GCP_CLUSTER_NAME=$(GCP_TF_CLUSTER_NAME) $(MAKE) gcloud-auth-cluster

# Creates a cluster and install current version of Agones controller
# Set all necessary variables as `make install` does
# Unifies previous `make gcloud-test-cluster` and `make install` targets
gcloud-terraform-install: GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT ?= 4
gcloud-terraform-install: GCP_CLUSTER_NODEPOOL_MACHINETYPE ?= e2-standard-4
gcloud-terraform-install: GCP_CLUSTER_NODEPOOL_AUTOSCALE ?= false
gcloud-terraform-install: GCP_CLUSTER_NODEPOOL_MIN_NODECOUNT ?= 1
gcloud-terraform-install: GCP_CLUSTER_NODEPOOL_MAX_NODECOUNT ?= 5
gcloud-terraform-install: GCP_CLUSTER_NODEPOOL_WINDOWSINITIALNODECOUNT ?= 0
gcloud-terraform-install: GCP_CLUSTER_NODEPOOL_WINDOWSMACHINETYPE ?= e2-standard-4
gcloud-terraform-install: ALWAYS_PULL_SIDECAR := true
gcloud-terraform-install: IMAGE_PULL_POLICY := "Always"
gcloud-terraform-install: PING_SERVICE_TYPE := "LoadBalancer"
gcloud-terraform-install: CRD_CLEANUP := true
gcloud-terraform-install: GCP_TF_CLUSTER_NAME ?= agones-tf-cluster
gcloud-terraform-install: LOG_LEVEL ?= debug
gcloud-terraform-install: FEATURE_GATES := $(ALPHA_FEATURE_GATES)
gcloud-terraform-install: GCP_PROJECT ?= $(current_project)
gcloud-terraform-install:
	$(MAKE) terraform-init DIRECTORY=gke
	$(DOCKER_RUN) bash -c ' \
	cd $(mount_path)/build/terraform/gke && terraform apply -auto-approve -var agones_version="$(VERSION)" -var image_registry="$(REGISTRY)" \
		-var pull_policy="$(IMAGE_PULL_POLICY)" \
		-var always_pull_sidecar="$(ALWAYS_PULL_SIDECAR)" \
		-var image_pull_secret="$(IMAGE_PULL_SECRET)" \
		-var ping_service_type="$(PING_SERVICE_TYPE)" \
		-var crd_cleanup="$(CRD_CLEANUP)" \
		-var chart="../../../install/helm/agones/" \
		-var name=$(GCP_TF_CLUSTER_NAME) -var machine_type="$(GCP_CLUSTER_NODEPOOL_MACHINETYPE)" \
		-var project=$(GCP_PROJECT) \
		-var location=$(GCP_CLUSTER_LOCATION) \
		-var log_level=$(LOG_LEVEL) \
		-var autoscale=$(GCP_CLUSTER_NODEPOOL_AUTOSCALE) \
		-var min_node_count=$(GCP_CLUSTER_NODEPOOL_MIN_NODECOUNT) \
		-var max_node_count=$(GCP_CLUSTER_NODEPOOL_MAX_NODECOUNT) \
		-var feature_gates=$(FEATURE_GATES) \
		-var node_count=$(GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT) \
		-var windows_node_count=$(GCP_CLUSTER_NODEPOOL_WINDOWSINITIALNODECOUNT) \
		-var windows_machine_type=$(GCP_CLUSTER_NODEPOOL_WINDOWSMACHINETYPE)'
	GCP_CLUSTER_NAME=$(GCP_TF_CLUSTER_NAME) $(MAKE) gcloud-auth-cluster

gcloud-terraform-destroy-cluster: GCP_PROJECT ?= $(current_project)
gcloud-terraform-destroy-cluster:
	$(MAKE) terraform-init DIRECTORY=gke
	$(DOCKER_RUN) bash -c 'cd $(mount_path)/build/terraform/gke && terraform destroy -var project=$(GCP_PROJECT) -auto-approve'

terraform-test: $(ensure-build-image)
terraform-test: GCP_PROJECT ?= $(current_project)
terraform-test:
	$(MAKE) terraform-init TERRAFORM_BUILD_DIR=$(mount_path)/test/terraform
	$(MAKE) run-terraform-test GCP_PROJECT=$(GCP_PROJECT)
	$(MAKE) terraform-test-clean

terraform-test-clean: $(ensure-build-image)
	$(MAKE) terraform-clean TERRAFORM_BUILD_DIR=$(mount_path)/test/terraform

# run terratest which verifies GKE and Helm Terraform modules
run-terraform-test:
	$(DOCKER_RUN) bash -c 'cd $(mount_path)/test/terraform && go test -v -run TestTerraformGKEInstallConfig \
	-timeout 1h -project $(GCP_PROJECT) $(ARGS)'
