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


#   __  __ _       _ _          _
#  |  \/  (_)_ __ (_) | ___   _| |__   ___
#  | |\/| | | '_ \| | |/ / | | | '_ \ / _ \
#  | |  | | | | | | |   <| |_| | |_) |  __/
#  |_|  |_|_|_| |_|_|_|\_\\__,_|_.__/ \___|
#

# Switches to an "agones" profile, and starts a kubernetes cluster
# of the right version.
#
# Use MINIKUBE_DRIVER variable to change the VM driver
# (defaults virtualbox for Linux and macOS, hyperv for windows) if you so desire.
minikube-test-cluster: DOCKER_RUN_ARGS+=--network=host -v $(minikube_cert_mount)
minikube-test-cluster: $(ensure-build-image) minikube-agones-profile
	$(MINIKUBE) start --kubernetes-version v1.14.10 --vm-driver $(MINIKUBE_DRIVER)
	# wait until the master is up
	until docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) kubectl cluster-info; \
		do \
			echo "Waiting for cluster to start..."; \
			sleep 1; \
		done
	$(MAKE) setup-test-cluster DOCKER_RUN_ARGS="$(DOCKER_RUN_ARGS)"
	$(MAKE) minikube-post-start

# switch to the agones cluster
minikube-agones-profile:
	$(MINIKUBE) profile $(MINIKUBE_PROFILE)

# Connecting to minikube requires so enhanced permissions, so use this target
# instead of `make shell` to start an interactive shell for development on minikube.
minikube-shell: $(ensure-build-image) minikube-agones-profile
	$(MAKE) shell DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount) $(DOCKER_RUN_ARGS)"

# Push the local Agones Docker images that have already been built
# via `make build` or `make build-images` into the "agones" minikube instance.
minikube-push: minikube-agones-profile
	$(MAKE) minikube-transfer-image TAG=$(sidecar_tag)
	$(MAKE) minikube-transfer-image TAG=$(controller_tag)
	$(MAKE) minikube-transfer-image TAG=$(ping_tag)
	$(MAKE) minikube-transfer-image TAG=$(allocator_tag)

# Installs the current development version of Agones into the Kubernetes cluster.
# Use this instead of `make install`, as it disables PullAlways on the install.yaml
minikube-install: minikube-agones-profile
	$(MAKE) install DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)" ALWAYS_PULL_SIDECAR=false \
		IMAGE_PULL_POLICY=IfNotPresent PING_SERVICE_TYPE=NodePort ALLOCATOR_SERVICE_TYPE=NodePort

minikube-uninstall: $(ensure-build-image) minikube-agones-profile
	$(MAKE) uninstall DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"

# Convenience target for transferring images into minikube.
# Use TAG to specify the image to transfer into minikube
minikube-transfer-image:
	docker save $(TAG) | ($(MINIKUBE_DOCKER_ENV) && docker load)

# Runs e2e tests against our minikube
minikube-test-e2e: DOCKER_RUN_ARGS=--network=host -v $(minikube_cert_mount)
minikube-test-e2e: minikube-agones-profile test-e2e

# Runs stress tests against our minikube
minikube-stress-test-e2e: DOCKER_RUN_ARGS=--network=host -v $(minikube_cert_mount)
minikube-stress-test-e2e: minikube-agones-profile stress-test-e2e

# prometheus on minkube
# we have to disable PVC as it's not supported on minkube.
minikube-setup-prometheus:
	$(MAKE) setup-prometheus \
		DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)" \
		PVC=false HELM_ARGS="--set server.resources.requests.cpu=0,server.resources.requests.memory=0"


# grafana on minkube with dashboards and prometheus datasource installed.
# we have to disable PVC as it's not supported on minkube.
minikube-setup-grafana:
	$(MAKE) setup-grafana \
		DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"

# minikube port forwarding
minikube-controller-portforward:
	$(MAKE) controller-portforward DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"

# minikube port forwarding grafana
minikube-grafana-portforward:
	$(MAKE) grafana-portforward \
		DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"

# minikube port forwarding for prometheus web ui
minikube-prometheus-portforward:
	$(MAKE) prometheus-portforward \
		DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"
