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

MINIKUBE_DRIVER ?= docker

# minikube shell mount for certificates
minikube_cert_mount := ~/.minikube:$(HOME)/.minikube

#   __  __ _       _ _          _
#  |  \/  (_)_ __ (_) | ___   _| |__   ___
#  | |\/| | | '_ \| | |/ / | | | '_ \ / _ \
#  | |  | | | | | | |   <| |_| | |_) |  __/
#  |_|  |_|_|_| |_|_|_|\_\\__,_|_.__/ \___|
#

# Switches to an "agones" profile, and starts a kubernetes cluster
# of the right version.
minikube-test-cluster: DOCKER_RUN_ARGS+=--network=host -v $(minikube_cert_mount)
minikube-test-cluster: $(ensure-build-image)
	$(MINIKUBE) start --kubernetes-version v1.29.7 -p $(MINIKUBE_PROFILE) --driver $(MINIKUBE_DRIVER)

# Connecting to minikube requires so enhanced permissions, so use this target
# instead of `make shell` to start an interactive shell for development on minikube.
minikube-shell: $(ensure-build-image)
	$(MAKE) shell DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount) $(DOCKER_RUN_ARGS)"

# Push the local Agones Docker images that have already been built
# via `make build` or `make build-images` into the "agones" minikube instance.
minikube-push:
	$(MINIKUBE) image load $(sidecar_linux_amd64_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image load $(controller_amd64_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image load $(ping_amd64_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image load $(allocator_amd64_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image load $(extensions_amd64_tag) -p $(MINIKUBE_PROFILE)

	$(MINIKUBE) image tag $(sidecar_linux_amd64_tag) $(sidecar_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image tag $(controller_amd64_tag) $(controller_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image tag $(ping_amd64_tag) $(ping_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image tag $(allocator_amd64_tag) $(allocator_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image tag $(extensions_amd64_tag) $(extensions_tag) -p $(MINIKUBE_PROFILE)

# Installs the current development version of Agones into the Kubernetes cluster.
# Use this instead of `make install`, as it disables PullAlways on the install.yaml
minikube-install:
	$(MAKE) install DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)" ALWAYS_PULL_SIDECAR=false \
		IMAGE_PULL_POLICY=IfNotPresent PING_SERVICE_TYPE=NodePort ALLOCATOR_SERVICE_TYPE=NodePort

minikube-uninstall: $(ensure-build-image)
	$(MAKE) uninstall DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"

# Runs e2e tests against our minikube
minikube-test-e2e: DOCKER_RUN_ARGS=--network=host -v $(minikube_cert_mount)
minikube-test-e2e: test-e2e

# Runs stress tests against our minikube
minikube-stress-test-e2e: DOCKER_RUN_ARGS=--network=host -v $(minikube_cert_mount)
minikube-stress-test-e2e: stress-test-e2e

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

# prometheus-stack on minkube
# we have to disable PVC as it's not supported on minkube.
minikube-setup-prometheus-stack:
	$(MAKE) setup-prometheus-stack \
		DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)" \
		PVC=false HELM_ARGS="--set prometheus.server.resources.requests.cpu=0,prometheus.server.resources.requests.memory=0"

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
