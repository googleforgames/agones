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


#   _  ___           _
#  | |/ (_)_ __   __| |
#  | ' /| | '_ \ / _` |
#  | . \| | | | | (_| |
#  |_|\_\_|_| |_|\__,_|

# creates a kind cluster for use with agones
# Kind stand for Kubernetes IN Docker
# You can change the cluster name using KIND_PROFILE env var
kind-test-cluster: DOCKER_RUN_ARGS+=--network=host
kind-test-cluster: $(ensure-build-image)
	@if [ -z $$(kind get clusters | grep $(KIND_PROFILE)) ]; then\
		echo "Could not find $(KIND_PROFILE) cluster. Creating...";\
		kind create cluster --name $(KIND_PROFILE) --image kindest/node:v1.23.6 --wait 5m;\
	fi

# deletes the kind agones cluster
# useful if you're want to start from scratch
kind-delete-cluster:
	kind delete cluster --name $(KIND_PROFILE)

# start an interactive shell with kubectl configured to target the kind cluster
kind-shell: $(ensure-build-image)
	$(MAKE) shell DOCKER_RUN_ARGS="--network=host $(DOCKER_RUN_ARGS)"

# installs the current dev version of agones
# you should build-images and kind-push first.
kind-install:
	$(MAKE) install DOCKER_RUN_ARGS="--network=host" ALWAYS_PULL_SIDECAR=false \
		IMAGE_PULL_POLICY=IfNotPresent PING_SERVICE_TYPE=NodePort ALLOCATOR_SERVICE_TYPE=NodePort

# pushes the current dev version of agones to the kind single node cluster.
kind-push:
	kind load docker-image $(sidecar_linux_amd64_tag) --name="$(KIND_PROFILE)"
	kind load docker-image $(controller_tag) --name="$(KIND_PROFILE)"
	kind load docker-image $(ping_tag) --name="$(KIND_PROFILE)"
	kind load docker-image $(allocator_tag) --name="$(KIND_PROFILE)"

# Runs e2e tests against our kind cluster
kind-test-e2e:
	$(MAKE) DOCKER_RUN_ARGS=--network=host test-e2e

# prometheus on kind
# we have to disable PVC as it's not supported on kind.
kind-setup-prometheus:
	$(MAKE) setup-prometheus DOCKER_RUN_ARGS="--network=host" PVC=false \
		HELM_ARGS="--set server.resources.requests.cpu=0,server.resources.requests.memory=0"

# grafana on kind with dashboards and prometheus datasource installed.
# we have to disable PVC as it's not supported on kind.
kind-setup-grafana:
	$(MAKE) setup-grafana DOCKER_RUN_ARGS="--network=host" PVC=false

kind-setup-prometheus-stack:
	$(MAKE) setup-prometheus-stack DOCKER_RUN_ARGS="--network=host" PVC=false \
		HELM_ARGS="--set prometheus.server.resources.requests.cpu=0,prometheus.server.resources.requests.memory=0"

# kind port forwarding controller web
kind-controller-portforward:
	$(MAKE) controller-portforward DOCKER_RUN_ARGS="--network=host"

# kind port forwarding grafana
kind-grafana-portforward:
	$(MAKE) grafana-portforward DOCKER_RUN_ARGS="--network=host"

# kind port forwarding for prometheus web ui
kind-prometheus-portforward:
	$(MAKE) prometheus-portforward DOCKER_RUN_ARGS="--network=host"
