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

### Deploy cluster with Terraform
terraform-init:
terraform-init: $(ensure-build-image)
	docker run --rm -it $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) bash -c '\
	cd $(mount_path)/install/terraform && terraform init && gcloud auth application-default login'

terraform-clean:
	rm -r ../install/terraform/.terraform
	rm ../install/terraform/terraform.tfstate*

# Creates a cluster and install release version of Agones controller
# Version could be specified by AGONES_VERSION
gcloud-terraform-cluster: GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT ?= 4
gcloud-terraform-cluster: GCP_CLUSTER_NODEPOOL_MACHINETYPE ?= n1-standard-4
gcloud-terraform-cluster: AGONES_VERSION ?= ''
gcloud-terraform-cluster: $(ensure-build-image)
gcloud-terraform-cluster:
ifndef GCP_PROJECT
	$(eval GCP_PROJECT=$(shell sh -c "gcloud config get-value project 2> /dev/null"))
endif
	$(DOCKER_RUN) bash -c 'export TF_VAR_agones_version=$(AGONES_VERSION) && \
		cd $(mount_path)/install/terraform && terraform apply -auto-approve -var values_file="" \
		-var chart="agones" \
	 	-var "cluster={name=\"$(GCP_CLUSTER_NAME)\", machineType=\"$(GCP_CLUSTER_NODEPOOL_MACHINETYPE)\", \
		 zone=\"$(GCP_CLUSTER_ZONE)\", project=\"$(GCP_PROJECT)\", \
		 initialNodeCount=\"$(GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT)\"}"'
	$(MAKE) gcloud-auth-cluster

# Creates a cluster and install current version of Agones controller
# Set all necessary variables as `make install` does
gcloud-terraform-install: GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT ?= 4
gcloud-terraform-install: GCP_CLUSTER_NODEPOOL_MACHINETYPE ?= n1-standard-4
gcloud-terraform-install: ALWAYS_PULL_SIDECAR := true
gcloud-terraform-install: IMAGE_PULL_POLICY := "Always"
gcloud-terraform-install: PING_SERVICE_TYPE := "LoadBalancer"
gcloud-terraform-install: CRD_CLEANUP := true
gcloud-terraform-install:
ifndef GCP_PROJECT
	$(eval GCP_PROJECT=$(shell sh -c "gcloud config get-value project 2> /dev/null"))
endif
	$(DOCKER_RUN) bash -c ' \
		cd $(mount_path)/install/terraform && terraform apply -auto-approve -var agones_version="$(VERSION)" -var image_registry="$(REGISTRY)" \
		-var pull_policy="$(IMAGE_PULL_POLICY)" \
		-var always_pull_sidecar="$(ALWAYS_PULL_SIDECAR)" \
		-var image_pull_secret="$(IMAGE_PULL_SECRET)" \
		-var ping_service_type="$(PING_SERVICE_TYPE)" \
		-var crd_cleanup="$(CRD_CLEANUP)" \
		-var "cluster={name=\"$(GCP_CLUSTER_NAME)\", machineType=\"$(GCP_CLUSTER_NODEPOOL_MACHINETYPE)\", \
		 zone=\"$(GCP_CLUSTER_ZONE)\", project=\"$(GCP_PROJECT)\", \
		 initialNodeCount=\"$(GCP_CLUSTER_NODEPOOL_INITIALNODECOUNT)\"}"'
	$(MAKE) gcloud-auth-cluster

gcloud-terraform-destroy-cluster:
	$(DOCKER_RUN) bash -c 'cd $(mount_path)/install/terraform && \
	 terraform destroy -auto-approve'
