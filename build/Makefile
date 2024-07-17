# Copyright 2017 Google LLC All Rights Reserved.
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
# Makefile for building, testing and developing Agones
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

# base version target. This is usually the next release.
base_version = 1.43.0

#
# All of the following can be overwritten with environment variables
# or passed through directly when invoking the relevent Make targets
#

# Version defaults to the short hash of the latest commit
VERSION ?= $(base_version)-dev-$(shell git rev-parse --short=7 HEAD)

# kubectl configuration to use
KUBECONFIG ?= ~/.kube/config
# The (gcloud) test cluster that is being worked against
GCP_CLUSTER_NAME ?= test-cluster
GCP_CLUSTER_ZONE ?= us-west1-c
GCP_CLUSTER_LOCATION ?= $(GCP_CLUSTER_ZONE)
GCP_BUCKET_CHARTS ?= agones-chart
# the profile to use when developing on minikube
MINIKUBE_PROFILE ?= agones
GO_BUILD_TAGS ?= none
BUILDX_WINDOWS_BUILDER = windows-builder
BUILDX_ARM64_BUILDER = arm64-builder
WINDOWS_DOCKER_PUSH_ARGS =
# Versions of Windows Server to support, default is Windows Server 2019.
# For the full list of tags see https://hub.docker.com/_/microsoft-windows-servercore.
# GKE-Windows version map: https://cloud.google.com/kubernetes-engine/docs/how-to/creating-a-cluster-windows#version_mapping
WINDOWS_VERSIONS = ltsc2019 ltsc2022

# https://github.com/microsoft/Windows-Containers/issues/493: Pin the LTSC2019 image to an older sha
LTSC2019_IMAGE_TAG = "@sha256:6fdf140282a2f809dae9b13fe441635867f0a27c33a438771673b8da8f3348a4"

# Specify stress test level 1..100
# STRESS_TEST_LEVEL=n requires capacity between 50*n up to 100*n simple-game-server Game Servers.
STRESS_TEST_LEVEL ?= 20

# kind cluster name to use
KIND_PROFILE ?= agones
KIND_CONTAINER_NAME=$(KIND_PROFILE)-control-plane

# Game Server image to use while doing end-to-end tests
GS_TEST_IMAGE ?= us-docker.pkg.dev/agones-images/examples/simple-game-server:0.34

# Enable all beta feature gates. Keep in sync with `true` (beta) entries in pkg/util/runtime/features.go:featureDefaults
BETA_FEATURE_GATES ?= "CountsAndLists=true&DisableResyncOnSDKServer=true"

# Enable all alpha feature gates. Keep in sync with `false` (alpha) entries in pkg/util/runtime/features.go:featureDefaults
ALPHA_FEATURE_GATES ?= "PlayerAllocationFilter=true&PlayerTracking=true&RollingUpdateFix=true&PortRanges=true&PortPolicyNone=true&AutopilotPassthroughPort=true&ScheduledAutoscaler=true&Example=true"

# Build with Windows support
WITH_WINDOWS=1
# Build with arm64support
WITH_ARM64=1

# Directory that this Makefile is in.
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
build_path := $(dir $(mkfile_path))
agones_path := $(realpath $(build_path)/..)
site_path := $(agones_path)/site

kubeconfig_path := $(dir $(KUBECONFIG))
kubeconfig_file := $(notdir $(KUBECONFIG))

helm_config := ~/.config/helm
helm_cache := ~/.cache/helm

agones_package = agones.dev/agones
mount_path = /go/src/$(agones_package)
common_mounts = -v $(build_path)/.config/gcloud:/root/.config/gcloud \
				-v $(kubeconfig_path):/root/.kube \
				-v $(helm_config):/root/.config/helm \
				-v $(helm_cache):/root/.cache/helm \
				-v $(agones_path):$(mount_path) \
				-v $(build_path)/.gomod:/go/pkg/mod

# Its possible for this to change in the future, so it gets its own variable.
REGISTRY ?= local
workdir_path = $(mount_path)

build_version = $(call sha,$(build_path)/build-image/Dockerfile)
build_tag = agones-build:$(build_version)
controller_tag = $(REGISTRY)/agones-controller:$(VERSION)
extensions_tag = $(REGISTRY)/agones-extensions:$(VERSION)
sidecar_tag = $(REGISTRY)/agones-sdk:$(VERSION)
allocator_tag = $(REGISTRY)/agones-allocator:$(VERSION)
ping_tag = $(REGISTRY)/agones-ping:$(VERSION)

ifeq ($(WITH_WINDOWS), 1)
sidecar_linux_amd64_tag = $(REGISTRY)/agones-sdk:$(VERSION)-linux-amd64
sidecar_linux_arm64_tag = $(REGISTRY)/agones-sdk:$(VERSION)-linux-arm64
else
sidecar_linux_amd64_tag = $(REGISTRY)/agones-sdk:$(VERSION)-amd64
sidecar_linux_arm64_tag = $(REGISTRY)/agones-sdk:$(VERSION)-arm64
endif

allocator_amd64_tag = $(allocator_tag)-amd64
allocator_arm64_tag = $(allocator_tag)-arm64
controller_amd64_tag = $(controller_tag)-amd64
controller_arm64_tag = $(controller_tag)-arm64
extensions_amd64_tag = $(extensions_tag)-amd64
extensions_arm64_tag = $(extensions_tag)-arm64
ping_amd64_tag = $(ping_tag)-amd64
ping_arm64_tag = $(ping_tag)-arm64

# build list of images to include in the manifest for the sidecar image.
push_sidecar_manifest = $(sidecar_linux_amd64_tag)
push_allocator_manifest = $(allocator_amd64_tag)
push_controller_manifest = $(controller_amd64_tag)
push_extensions_manifest = $(extensions_amd64_tag)
push_ping_manifest = $(ping_amd64_tag)
gomod_on = GO111MODULE=on

go_version_flags = -ldflags "-X agones.dev/agones/pkg.Version=$(VERSION)"
DOCKER_RUN ?= docker run --rm $(common_mounts) -e "KUBECONFIG=/root/.kube/$(kubeconfig_file)" -e "$(gomod_on)" -w $(workdir_path) $(DOCKER_RUN_ARGS) $(build_tag)

ifdef DOCKER_RUN
	ensure-build-image += ensure-build-image
endif

git_safe = git config --global --add safe.directory $(mount_path)

# When performing full build, always rebuild go packages and compress more
ifdef FULL_BUILD
	go_rebuild_flags = -a
	zip_flags = -9
endif

ifndef FULL_BUILD
	# keep a cache of files built by Go across docker invocations in a local directory.
	common_mounts += -v $(build_path)/.gocache:/root/.cache/go-build
	go_rebuild_flags =

	# do not compress files in zips, slightly faster
	zip_flags = -0
endif

go_build_cmd = go build -mod=vendor -buildvcs=false
GO_BUILD_LINUX_AMD64=docker run --rm -e "CGO_ENABLED=0" -e "$(gomod_on)" -w $(workdir_path) $(common_mounts) $(build_tag) $(go_build_cmd)
GO_BUILD_LINUX_ARM64=docker run --rm -e "CGO_ENABLED=0" -e "GOOS=linux" -e "GOARCH=arm64" -e "$(gomod_on)" -w $(workdir_path) $(common_mounts) $(build_tag) $(go_build_cmd)
GO_BUILD_DARWIN_AMD64=docker run --rm -e "GOOS=darwin" -e "GOARCH=amd64" -e "$(gomod_on)" -w $(workdir_path) $(common_mounts) $(build_tag) $(go_build_cmd)
GO_BUILD_DARWIN_ARM64=docker run --rm -e "GOOS=darwin" -e "GOARCH=arm64" -e "$(gomod_on)" -w $(workdir_path) $(common_mounts) $(build_tag) $(go_build_cmd)
GO_BUILD_WINDOWS_AMD64=docker run --rm -e "GOOS=windows" -e "GOARCH=amd64" -e "$(gomod_on)" -w $(workdir_path) $(common_mounts) $(build_tag) $(go_build_cmd)

RACE_DETECTOR_ARGS=-race

ifdef NO_RACE_DETECTOR
  RACE_DETECTOR_ARGS=
endif

go_test_args = -mod=vendor $(RACE_DETECTOR_ARGS)
go_test_cmd = go test $(go_test_args)
GO_TEST=$(DOCKER_RUN) $(go_test_cmd)
GO_E2E_TEST_ARGS=--kubeconfig /root/.kube/$(kubeconfig_file)

gotestsum_json=/tmp/agones.gotestsum.json
ifdef GOTESTSUM_VERBOSE
gotestsum_format=--format=testname \
  --post-run-command="sh -c 'echo; echo --- RAW VERBOSE OUTPUT ---; jq -j \"select(.Output != null) | (\\\"VERBOSE: \\\" + .Output)\" < $(gotestsum_json); echo --- END RAW VERBOSE OUTPUT ---'"
else
gotestsum_format=--format=standard-quiet
endif
gotestsum_cmd=gotestsum $(gotestsum_format) --jsonfile=$(gotestsum_json) --hide-summary=skipped --rerun-fails=2 $(GOTESTSUM_ARGS)
GOTESTSUM=$(DOCKER_RUN) $(gotestsum_cmd)

PERF_OUTPUT_DIR=$(mount_path)/build/.perf

go_build_base_path=$(mount_path)

ifdef LOCAL_GO
	# use local Go tooling, which greatly speeds up incremental rebuilds, in particular on macOS
	# Note that agones.dev must be in GOPATH
	GO_BUILD_LINUX_AMD64=CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(gomod_on) $(go_build_cmd)
	GO_BUILD_WINDOWS_AMD64=GOOS=windows GOARCH=amd64 $(gomod_on) $(go_build_cmd)
	GO_BUILD_DARWIN_AMD64=GOOS=darwin GOARCH=amd64 $(gomod_on) $(go_build_cmd)
	GO_TEST=$(go_test_cmd) -v
	GOTESTSUM=$(gotestsum_cmd)
	GO_E2E_TEST_ARGS=
	PERF_OUTPUT_DIR=$(build_path)/.perf
	go_build_base_path=$(agones_path)
endif

ZIP_SDK=docker run --rm $(common_mounts) -w $(workdir_path)/cmd/sdk-server/bin/ $(build_tag) zip $(zip_flags)
ifdef LOCAL_ZIP
	ZIP_SDK=cd $(agones_path)/cmd/sdk-server/bin && zip -0
endif

unameprocessor := $(shell uname -p)
ifeq ($(unameprocessor),arm)
	DOCKER_BUILD_ARGS += --platform linux/x86_64
	DOCKER_RUN_ARGS += --platform linux/x86_64
endif

#    ___  ____    ___            _           _
#   / _ \/ ___|  |_ _|_ __   ___| |_   _  __| | ___
#  | | | \___ \   | || '_ \ / __| | | | |/ _` |/ _ \
#  | |_| |___) |  | || | | | (__| | |_| | (_| |  __/
#   \___/|____/  |___|_| |_|\___|_|\__,_|\__,_|\___|
#

uname := $(shell uname -s)
ifneq ($(findstring Microsoft,$(shell uname -r)),)
	osinclude := windows.mk
else ifeq ($(uname),Linux)
	osinclude := linux.mk
else ifeq ($(uname),Darwin)
	osinclude := macos.mk
endif

include ./includes/$(osinclude)

#   ___            _           _
#  |_ _|_ __   ___| |_   _  __| | ___ ___
#   | || '_ \ / __| | | | |/ _` |/ _ \ __|
#   | || | | | (__| | |_| | (_| |  __\__ \
#  |___|_| |_|\___|_|\__,_|\__,_|\___|___/
#

# personal includes, excluded from the git repository
-include ./local-includes/*.mk

include ./includes/build-image.mk
include ./includes/release.mk
include ./includes/google-cloud.mk
include ./includes/terraform.mk
include ./includes/minikube.mk
include ./includes/kind.mk
include ./includes/website.mk
include ./includes/sdk.mk
include ./includes/allocation.mk
include ./includes/examples.mk

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# Dynamic variables that impact targets below.
# Do not move as these need to be evaluated after the `local-includes` folder to allow for local override.

ifeq ($(WITH_WINDOWS), 1)
push_sidecar_manifest += $(foreach windows_version, $(WINDOWS_VERSIONS), $(sidecar_tag)-windows_amd64-$(windows_version))
endif
ifeq ($(WITH_ARM64), 1)
push_sidecar_manifest += $(sidecar_linux_arm64_tag)
push_allocator_manifest += $(allocator_arm64_tag)
push_controller_manifest += $(controller_arm64_tag)
push_extensions_manifest += $(extensions_arm64_tag)
push_ping_manifest += $(ping_arm64_tag)
endif

# build all
build: build-images build-sdks

# build the docker images
build-images: build-controller-image build-extensions-image build-agones-sdk-image build-ping-image build-allocator-image

# package the current agones helm chart
build-chart: RELEASE_VERSION ?= $(base_version)
build-chart: CHART_DIR ?= install/helm/agones/
build-chart: $(ensure-build-image)
	docker run --rm $(common_mounts) -w $(workdir_path) $(build_tag) bash -c \
		"mkdir -p install/helm/bin/ && rm -rf install/helm/bin/* && \
			helm package -d install/helm/bin/ --version $(RELEASE_VERSION) $(CHART_DIR)"

# push a specific release useful to push previous missing release
push-release-chart: RELEASE_VERSION ?= $(base_version)
push-release-chart: $(ensure-build-image)
	rm -rf /tmp/agones $(agones_path)/install/.helm-$(RELEASE_VERSION)/
	mkdir -p $(agones_path)/install/.helm-$(RELEASE_VERSION)/
	cd /tmp && git clone --single-branch -b release-$(RELEASE_VERSION) git@github.com:googleforgames/agones.git
	mv /tmp/agones/install/helm/agones $(agones_path)/install/.helm-$(RELEASE_VERSION)/
	CHART_DIR=install/.helm-$(RELEASE_VERSION)/agones $(MAKE) push-chart

# Run all tests
test: $(ensure-build-image) test-go test-sdks test-install-yaml site-test

# Run go tests
test-go: $(ensure-build-image)
	$(GO_TEST) $(agones_package)/pkg/... \
	    $(agones_package)/sdks/... $(agones_package)/cmd/...

# Runs end-to-end tests on the current configured cluster
# For minikube user the minikube-test-e2e targets
test-e2e:
	$(MAKE) DOCKER_RUN_ARGS="$(DOCKER_RUN_ARGS)" test-e2e-integration
	$(MAKE) DOCKER_RUN_ARGS="$(DOCKER_RUN_ARGS)" test-e2e-failure
	$(MAKE) DOCKER_RUN_ARGS="$(DOCKER_RUN_ARGS)" test-e2e-ha-extensions
	$(MAKE) DOCKER_RUN_ARGS="$(DOCKER_RUN_ARGS)" test-e2e-allocator-crash


# e2e test args:
# - Use ARGS to pass args to `go test` (but not the test binary itself)
# - Use GO_E2E_TEST_ARGS to pass args to the test binary
# - When E2E_USE_GOTESTSUM is enabled, use GOTESTSUM_ARGS to pass args to gotestsum
# If GAMESERVERS_NAMESPACE is empty - random namespace will be created
test-e2e-integration: FEATURE_GATES ?= $(ALPHA_FEATURE_GATES)
test-e2e-integration: CLOUD_PRODUCT ?= generic
test-e2e-integration: GAMESERVERS_NAMESPACE ?= ""
test-e2e-integration: GO_E2E_TEST_INTEGRATION_ARGS ?=\
  --cloud-product=$(CLOUD_PRODUCT) \
  --gameserver-image=$(GS_TEST_IMAGE) \
  --pullsecret=$(IMAGE_PULL_SECRET) \
  --feature-gates=$(FEATURE_GATES) \
  --namespace=$(GAMESERVERS_NAMESPACE)
test-e2e-integration: $(ensure-build-image)
	echo "Starting e2e integration test!"
ifdef E2E_USE_GOTESTSUM
	$(GOTESTSUM) --packages=$(agones_package)/test/e2e -- $(go_test_args) -timeout=30m $(ARGS) -args \
		$(GO_E2E_TEST_ARGS) $(GO_E2E_TEST_INTEGRATION_ARGS)
else
	$(GO_TEST) -timeout=30m $(ARGS) $(agones_package)/test/e2e -args \
	    $(GO_E2E_TEST_ARGS) $(GO_E2E_TEST_INTEGRATION_ARGS)
endif
	echo "Finishing e2e integration test!"

test-e2e-failure: CLOUD_PRODUCT ?= generic
test-e2e-failure: GO_E2E_TEST_INTEGRATION_ARGS ?=\
	--cloud-product=$(CLOUD_PRODUCT) \
	--gameserver-image=$(GS_TEST_IMAGE) \
	--pullsecret=$(IMAGE_PULL_SECRET)
test-e2e-failure: $(ensure-build-image)
	echo "Starting e2e controller failure test!"
ifdef E2E_USE_GOTESTSUM
	$(GOTESTSUM) --packages=$(agones_package)/test/e2e/controller -- $(go_test_args) $(ARGS) -parallel=1 -args \
		$(GO_E2E_TEST_ARGS) $(GO_E2E_TEST_INTEGRATION_ARGS)
else
	$(GO_TEST) $(ARGS) -parallel=1 $(agones_package)/test/e2e/controller -args \
	    $(GO_E2E_TEST_ARGS) $(GO_E2E_TEST_INTEGRATION_ARGS)
endif
	echo "Finishing e2e controller failure test!"

test-e2e-ha-extensions: FEATURE_GATES ?= $(ALPHA_FEATURE_GATES)
test-e2e-ha-extensions: CLOUD_PRODUCT ?= generic
test-e2e-ha-extensions: GO_E2E_TEST_INTEGRATION_ARGS ?=\
	--cloud-product=$(CLOUD_PRODUCT) \
	--gameserver-image=$(GS_TEST_IMAGE) \
	--feature-gates=$(FEATURE_GATES) \
	--pullsecret=$(IMAGE_PULL_SECRET)
test-e2e-ha-extensions: $(ensure-build-image)
	echo "Starting e2e extensions high availability test!"
ifdef E2E_USE_GOTESTSUM
	$(GOTESTSUM) --packages=$(agones_package)/test/e2e/extensions -- $(go_test_args) $(ARGS) -parallel=1 -args \
		$(GO_E2E_TEST_ARGS) $(GO_E2E_TEST_INTEGRATION_ARGS)
else
	$(GO_TEST) $(ARGS) -parallel=1 $(agones_package)/test/e2e/extensions -args \
	    $(GO_E2E_TEST_ARGS) $(GO_E2E_TEST_INTEGRATION_ARGS)
endif
	echo "Finishing e2e extensions high availability test!"

test-e2e-allocator-crash: FEATURE_GATES ?= $(ALPHA_FEATURE_GATES)
test-e2e-allocator-crash: CLOUD_PRODUCT ?= generic
test-e2e-allocator-crash: GO_E2E_TEST_INTEGRATION_ARGS ?=\
	--cloud-product=$(CLOUD_PRODUCT) \
	--gameserver-image=$(GS_TEST_IMAGE) \
	--feature-gates=$(FEATURE_GATES) \
	--pullsecret=$(IMAGE_PULL_SECRET)
test-e2e-allocator-crash: $(ensure-build-image)
	echo "Starting e2e allocation pod deletion test!"
ifdef E2E_USE_GOTESTSUM
	$(GOTESTSUM) --packages=$(agones_package)/test/e2e/allocator -- $(go_test_args) $(ARGS) -parallel=1 -args \
		$(GO_E2E_TEST_ARGS) $(GO_E2E_TEST_INTEGRATION_ARGS)
else
	$(GO_TEST) $(ARGS) -parallel=1 $(agones_package)/test/e2e/allocator -args \
	    $(GO_E2E_TEST_ARGS) $(GO_E2E_TEST_INTEGRATION_ARGS)
endif
	echo "Finishing e2e allocation pod deletion test!"

# Runs end-to-end stress tests on the current configured cluster
# For minikube user the minikube-stress-test-e2e targets
stress-test-e2e: $(ensure-build-image)
	$(GO_TEST) $(agones_package)/test/e2e $(ARGS) \
		-timeout 1h \
		-run '.*StressTest.*' \
		-args \
		$(GO_E2E_TEST_ARGS) \
		--gameserver-image=$(GS_TEST_IMAGE) \
		--pullsecret=$(IMAGE_PULL_SECRET) \
		--stress $(STRESS_TEST_LEVEL) \
		--perf-output $(PERF_OUTPUT_DIR) \
		--version $(VERSION)

# Run test on install yaml - make sure there is no change
# mostly this is for CI
test-install-yaml:
	-mkdir -p /tmp/agones-install
	cp $(agones_path)/install/yaml/install.yaml /tmp/agones-install/install.yaml
	sort /tmp/agones-install/install.yaml > /tmp/agones-install/install.yaml.sorted
	$(MAKE) gen-install
	sort $(agones_path)/install/yaml/install.yaml > /tmp/agones-install/install.current.yaml.sorted
	diff /tmp/agones-install/install.yaml.sorted /tmp/agones-install/install.current.yaml.sorted

# Push all the images up to $(REGISTRY)
push: push-controller-image push-extensions-image push-agones-sdk-image push-ping-image push-allocator-image

# Installs the current development version of Agones into the Kubernetes cluster
install: ALWAYS_PULL_SIDECAR := true
install: IMAGE_PULL_POLICY := "Always"
install: PING_SERVICE_TYPE := "LoadBalancer"
install: ALLOCATOR_SERVICE_TYPE := "LoadBalancer"
install: CRD_CLEANUP := true
install: LOG_LEVEL := "debug"
install: EXTERNAL_IP ?= $(shell $(DOCKER_RUN) kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
install: FEATURE_GATES ?= $(ALPHA_FEATURE_GATES)
install: SERVICE_MONITOR := false
install: HELM_ARGS ?=
install: $(ensure-build-image) install-custom-pull-secret
	-$(DOCKER_RUN) bash -c '[[ $$(helm status agones -n agones-system --output json | jq -r ".info.status") =~ (failed|pending-.*) ]] && helm uninstall agones --namespace=agones-system || true'
	$(DOCKER_RUN) \
		helm upgrade --install --atomic --wait --timeout 10m --namespace=agones-system \
		--create-namespace \
		--set agones.image.tag=$(VERSION),agones.image.registry=$(REGISTRY) \
		--set agones.image.controller.pullPolicy=$(IMAGE_PULL_POLICY),agones.image.controller.pullSecret=$(IMAGE_PULL_SECRET) \
		--set agones.image.extensions.pullPolicy=$(IMAGE_PULL_POLICY),agones.image.allocator.pullPolicy=$(IMAGE_PULL_POLICY) \
		--set agones.image.ping.pullPolicy=$(IMAGE_PULL_POLICY),agones.image.sdk.alwaysPull=$(ALWAYS_PULL_SIDECAR) \
		--set agones.ping.http.serviceType=$(PING_SERVICE_TYPE),agones.ping.udp.serviceType=$(PING_SERVICE_TYPE) \
		--set agones.allocator.service.serviceType=$(ALLOCATOR_SERVICE_TYPE) \
		--set agones.controller.logLevel=$(LOG_LEVEL) \
		--set agones.crds.cleanupOnDelete=$(CRD_CLEANUP) \
		--set agones.featureGates=$(FEATURE_GATES) \
		--set agones.allocator.service.loadBalancerIP=$(EXTERNAL_IP) \
		--set agones.metrics.serviceMonitor.enabled=$(SERVICE_MONITOR) \
		$(HELM_ARGS) \
		agones $(mount_path)/install/helm/agones/

update-allocation-certs: NAMESPACE ?= default
update-allocation-certs: install
update-allocation-certs:
	-mkdir -p $(agones_path)/build/allocation
	$(DOCKER_RUN) bash -c 'kubectl get secret allocator-client.default -n $(NAMESPACE) -ojsonpath="{.data.tls\.crt}" | base64 -d > $(mount_path)/build/allocation/client.crt'
	$(DOCKER_RUN) bash -c 'kubectl get secret allocator-client.default -n $(NAMESPACE) -ojsonpath="{.data.tls\.key}" | base64 -d > $(mount_path)/build/allocation/client.key'
	$(DOCKER_RUN) bash -c 'kubectl get secret allocator-tls-ca -n agones-system -ojsonpath="{.data.tls-ca\.crt}" | base64 -d > $(mount_path)/build/allocation/ca.crt'

uninstall: $(ensure-build-image)
	$(DOCKER_RUN) \
		helm uninstall agones --namespace=agones-system

# Build a static binary for the gameserver controller
ifeq ($(WITH_ARM64), 1)
build-controller-binary: $(ensure-build-image) build-controller-binary-linux-amd64 build-controller-binary-linux-arm64
else
build-controller-binary: $(ensure-build-image) build-controller-binary-linux-amd64
endif

build-controller-binary-linux-amd64: $(ensure-build-image)
	$(GO_BUILD_LINUX_AMD64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/controller/bin/controller.linux.amd64 \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/controller
build-controller-binary-linux-arm64: $(ensure-build-image)
	$(GO_BUILD_LINUX_ARM64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/controller/bin/controller.linux.arm64 \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/controller

# Build a static binary for the gamerserver extensions
ifeq ($(WITH_ARM64), 1)
build-extensions-binary: $(ensure-build-image) build-extensions-binary-linux-amd64 build-extensions-binary-linux-arm64
else
build-extensions-binary: $(ensure-build-image) build-extensions-binary-linux-amd64
endif

build-extensions-binary-linux-amd64: $(ensure-build-image)
	$(GO_BUILD_LINUX_AMD64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/extensions/bin/extensions.linux.amd64 \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/extensions
build-extensions-binary-linux-arm64: $(ensure-build-image)
	$(GO_BUILD_LINUX_ARM64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/extensions/bin/extensions.linux.arm64 \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/extensions

# Lint the go source code and yaml files.
# use LINT_TIMEOUT to manipulate the linter timeout
lint: LINT_TIMEOUT ?= 15m
lint: $(ensure-build-image)
	docker run -t -e "TERM=xterm-256color" -e "$(gomod_on)" --rm $(common_mounts) -w $(workdir_path) $(DOCKER_RUN_ARGS) $(build_tag) bash -c \
		"$(git_safe) && \
		golangci-lint run ./examples/... && \
		golangci-lint run --timeout $(LINT_TIMEOUT) ./... && \
		yamllint ."

# Extract licenses from source tree
build-licenses:
	docker run --rm $(common_mounts) $(build_tag) $(mount_path)/build/extract-licenses.sh

# Tarball source for dependencies that are required to be distributed with the image (MPL)
build-required-src-dist:
	docker run --rm $(common_mounts) $(build_tag) $(mount_path)/build/build-required-src-dist.sh

# Create docker builder used in creating arm64 images
# According to the docker documentation, to build the images for ARM64 arch CPU
# we would need to use buildx command which uses builder https://docs.docker.com/desktop/multi-arch/
# We primarily need this for M1 Macs, see: https://medium.com/geekculture/docker-build-with-mac-m1-d668c802ab96
# Making this command optional (using - as prefix) because it will fail once built for the first time, so we ignore failures.
create-arm64-builder:
	-docker buildx create --name $(BUILDX_ARM64_BUILDER)

# Build the image for the gameserver controller
build-controller-image: build-controller-image-amd64
ifeq ($(WITH_ARM64), 1)
build-controller-image: build-controller-image-arm64
endif

build-controller-image-amd64: $(ensure-build-image) build-controller-binary build-licenses build-required-src-dist
	docker build $(agones_path)/cmd/controller/ --tag=$(controller_amd64_tag) $(DOCKER_BUILD_ARGS)
# creating docker builder and then using that builder to build controller image in buildx command
build-controller-image-arm64: $(ensure-build-image) build-controller-binary create-arm64-builder
	docker buildx build --platform linux/arm64 --builder $(BUILDX_ARM64_BUILDER) --build-arg ARCH=arm64 $(agones_path)/cmd/controller/ --tag=$(controller_arm64_tag) $(DOCKER_BUILD_ARGS)

# push the gameservers controller image
push-controller-image: push-controller-image-amd64
ifeq ($(WITH_ARM64), 1)
push-controller-image: push-controller-image-arm64
endif
	# Docker >=24.0.6
	docker buildx imagetools create --tag $(controller_tag) $(push_controller_manifest)

push-controller-image-amd64: build-controller-image-amd64
	docker push $(controller_amd64_tag)
push-controller-image-arm64: build-controller-image-arm64
	$(MAKE) DOCKER_BUILD_ARGS=--push build-controller-image-arm64

# build the static binary for the gameserver sidecar
build-agones-sdk-binary: $(ensure-build-image) build-agones-sdk-binary-linux-amd64 build-agones-sdk-binary-linux-arm64 build-agones-sdk-binary-windows build-agones-sdk-binary-darwin-amd64 build-agones-sdk-binary-darwin-arm64
	$(ZIP_SDK) \
		agonessdk-server-$(VERSION).zip sdk-server.darwin.amd64 sdk-server.darwin.arm64 sdk-server.linux.amd64 sdk-server.linux.arm64 sdk-server.windows.amd64.exe

# build the static binary for the gameserver sidecar for Linux
build-agones-sdk-binary-linux-amd64: $(ensure-build-image)
	$(GO_BUILD_LINUX_AMD64) \
		-o $(go_build_base_path)/cmd/sdk-server/bin/sdk-server.linux.amd64 $(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/sdk-server
build-agones-sdk-binary-linux-arm64: $(ensure-build-image)
	$(GO_BUILD_LINUX_ARM64) \
		-o $(go_build_base_path)/cmd/sdk-server/bin/sdk-server.linux.arm64 $(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/sdk-server


# build the static binary for the gameserver sidecar for Darwin (macOS)
build-agones-sdk-binary-darwin-amd64: $(ensure-build-image)
	$(GO_BUILD_DARWIN_AMD64) \
		-o $(go_build_base_path)/cmd/sdk-server/bin/sdk-server.darwin.amd64 $(go_rebuild_flags) $(go_version_flags) $(agones_package)/cmd/sdk-server
build-agones-sdk-binary-darwin-arm64: $(ensure-build-image)
	$(GO_BUILD_DARWIN_ARM64) \
		-o $(go_build_base_path)/cmd/sdk-server/bin/sdk-server.darwin.arm64 $(go_rebuild_flags) $(go_version_flags) $(agones_package)/cmd/sdk-server

# build the windows binary for the gameserver sidecar for Windows
build-agones-sdk-binary-windows: $(ensure-build-image)
	$(GO_BUILD_WINDOWS_AMD64) \
		-o $(go_build_base_path)/cmd/sdk-server/bin/sdk-server.windows.amd64.exe $(go_rebuild_flags) $(go_version_flags) $(agones_package)/cmd/sdk-server

ensure-windows-buildx-%:
	-docker buildx create --name=$(BUILDX_WINDOWS_BUILDER)-$*

# Build the image for the gameserver extensions
build-extensions-image: build-extensions-image-amd64
ifeq ($(WITH_ARM64), 1)
build-extensions-image: build-extensions-image-arm64
endif

build-extensions-image-amd64: $(ensure-build-image) build-extensions-binary build-licenses build-required-src-dist
	docker build $(agones_path)/cmd/extensions/ --tag=$(extensions_amd64_tag) $(DOCKER_BUILD_ARGS)
# creating docker builder and then using that builder to build extensions image in buildx command
build-extensions-image-arm64: $(ensure-build-image) build-extensions-binary create-arm64-builder
	docker buildx build --platform linux/arm64 --builder $(BUILDX_ARM64_BUILDER) --build-arg ARCH=arm64 $(agones_path)/cmd/extensions/ --tag=$(extensions_arm64_tag) $(DOCKER_BUILD_ARGS)

# push the gameservers extensions image
push-extensions-image: push-extensions-image-amd64
ifeq ($(WITH_ARM64), 1)
push-extensions-image: push-extensions-image-arm64
endif
	# Docker >=24.0.6
	docker buildx imagetools create --tag $(extensions_tag) $(push_extensions_manifest)

# push extensions image
push-extensions-image-amd64: build-extensions-image-amd64
	docker push $(extensions_amd64_tag)
push-extensions-image-arm64: build-extensions-image-arm64
	$(MAKE) DOCKER_BUILD_ARGS=--push build-extensions-image-arm64

# Build the image for the gameserver sidecar and SDK binaries
build-agones-sdk-image: $(ensure-build-image) build-agones-sdk-binary build-licenses build-required-src-dist build-agones-sdk-image-amd64
ifeq ($(WITH_ARM64), 1)
build-agones-sdk-image: build-agones-sdk-image-arm64
endif
ifeq ($(WITH_WINDOWS), 1)
build-agones-sdk-image: build-agones-sdk-image-windows
endif

build-agones-sdk-image-amd64: $(ensure-build-image) build-agones-sdk-binary-linux-amd64
	docker build $(agones_path)/cmd/sdk-server/ --tag=$(sidecar_linux_amd64_tag) $(DOCKER_BUILD_ARGS)
# creating docker builder and then using that builder to build sdk image in buildx command
build-agones-sdk-image-arm64: $(ensure-build-image) build-agones-sdk-binary-linux-arm64 create-arm64-builder
	docker buildx build --platform linux/arm64 --builder $(BUILDX_ARM64_BUILDER) --build-arg ARCH=arm64 $(agones_path)/cmd/sdk-server/ --tag=$(sidecar_linux_arm64_tag) $(DOCKER_BUILD_ARGS)

build-agones-sdk-image-windows: build-agones-sdk-binary
build-agones-sdk-image-windows: $(foreach winver, $(WINDOWS_VERSIONS), build-agones-sdk-image-windows-$(winver))

# Build the image for the gameserver sidecar and SDK binaries
build-agones-sdk-image-windows-%: $(ensure-build-image) build-agones-sdk-binary build-licenses build-required-src-dist ensure-windows-buildx-%
	docker buildx build --platform windows/amd64 --builder $(BUILDX_WINDOWS_BUILDER)-$* -f $(agones_path)/cmd/sdk-server/Dockerfile.windows --tag=$(sidecar_tag)-windows_amd64-$* --build-arg WINDOWS_VERSION=$* --build-arg IMAGE_TAG=$(if $(filter ltsc2019,$*),$(LTSC2019_IMAGE_TAG)) $(DOCKER_BUILD_ARGS) $(agones_path)/cmd/sdk-server/ $(WINDOWS_DOCKER_PUSH_ARGS)

# Build a static binary for the ping service
build-ping-binary: build-ping-binary-amd64
ifeq ($(WITH_ARM64), 1)
build-ping-binary: build-ping-binary-arm64
endif

build-ping-binary-amd64: $(ensure-build-image)
	$(GO_BUILD_LINUX_AMD64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/ping/bin/ping.linux.amd64 \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/ping
build-ping-binary-arm64: $(ensure-build-image)
	$(GO_BUILD_LINUX_ARM64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/ping/bin/ping.linux.arm64 \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/ping

# Pushes up the ping image
push-ping-image: push-ping-image-amd64
ifeq ($(WITH_ARM64), 1)
push-ping-image: push-ping-image-arm64
endif
	# Docker >=24.0.6
	docker buildx imagetools create --tag $(ping_tag) $(push_ping_manifest)


push-ping-image-amd64: $(ensure-build-image) build-ping-image-amd64
	docker push $(ping_amd64_tag)
push-ping-image-arm64:
	$(MAKE) DOCKER_BUILD_ARGS=--push build-ping-image-arm64

# Build the image for the ping service
build-ping-image: build-ping-image-amd64
ifeq ($(WITH_ARM64), 1)
build-ping-image: build-ping-image-arm64
endif

build-ping-image-amd64: $(ensure-build-image) build-ping-binary build-licenses build-required-src-dist
	docker build $(agones_path)/cmd/ping/ --tag=$(ping_amd64_tag) $(DOCKER_BUILD_ARGS)
# creating docker builder and then using that builder to build ping image in buildx command
build-ping-image-arm64: $(ensure-build-image) build-ping-binary create-arm64-builder
	docker buildx build --platform linux/arm64 --builder $(BUILDX_ARM64_BUILDER) --build-arg ARCH=arm64 $(agones_path)/cmd/ping/ --tag=$(ping_arm64_tag) $(DOCKER_BUILD_ARGS)

# Build a static binary for the allocator service
build-allocator-binary: $(ensure-build-image) build-allocator-binary-linux-amd64
ifeq ($(WITH_ARM64), 1)
build-allocator-binary: build-allocator-binary-linux-arm64
endif

build-allocator-binary-linux-amd64: $(ensure-build-image)
	$(GO_BUILD_LINUX_AMD64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/allocator/bin/allocator.linux.amd64 \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/allocator
build-allocator-binary-linux-arm64:
	$(GO_BUILD_LINUX_ARM64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/allocator/bin/allocator.linux.arm64 \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/allocator

# Pushes up the allocator image
push-allocator-image: push-allocator-image-amd64
ifeq ($(WITH_ARM64), 1)
push-allocator-image: push-allocator-image-arm64
endif
	# Docker >=24.0.6
	docker buildx imagetools create --tag $(allocator_tag) $(push_allocator_manifest)

push-allocator-image-amd64: build-allocator-image-amd64
	docker push $(allocator_amd64_tag)
push-allocator-image-arm64: build-allocator-image-arm64
	$(MAKE) DOCKER_BUILD_ARGS=--push build-allocator-image-arm64

# Build the image for the allocator service
build-allocator-image: build-allocator-image-amd64
ifeq ($(WITH_ARM64), 1)
build-allocator-image: build-allocator-image-arm64
endif

build-allocator-image-amd64: $(ensure-build-image) build-allocator-binary build-licenses build-required-src-dist
	docker build $(agones_path)/cmd/allocator/ --tag=$(allocator_amd64_tag) $(DOCKER_BUILD_ARGS)
# creating docker builder and then using that builder to build allocator image in buildx command
build-allocator-image-arm64: $(ensure-build-image) build-allocator-binary create-arm64-builder
	docker buildx build $(agones_path)/cmd/allocator/ --builder $(BUILDX_ARM64_BUILDER) --tag=$(allocator_arm64_tag) --build-arg ARCH=arm64 --platform linux/arm64 $(DOCKER_BUILD_ARGS)

# push the gameservers sidecar image
push-agones-sdk-image: push-agones-sdk-linux-image-amd64
ifeq ($(WITH_WINDOWS), 1)
push-agones-sdk-image: push-agones-sdk-windows
endif
ifeq ($(WITH_ARM64), 1)
push-agones-sdk-image: push-agones-sdk-linux-image-arm64
endif
	# Docker >=24.0.6
	docker buildx imagetools create --tag $(sidecar_tag) $(push_sidecar_manifest)

push-agones-sdk-windows: WINDOWS_DOCKER_PUSH_ARGS=--push
push-agones-sdk-windows: build-agones-sdk-image-windows

push-agones-sdk-linux-image-arm64: $(ensure-build-image)
	$(MAKE) DOCKER_BUILD_ARGS=--push build-agones-sdk-image-arm64

push-agones-sdk-linux-image-amd64: build-agones-sdk-image-amd64
	docker push $(sidecar_linux_amd64_tag)

# Generate the Embedded CRD openapi
gen-embedded-openapi: ensure-build-image
	docker run --rm $(common_mounts) --workdir=$(mount_path)/build/scripts/k8s-export-openapi $(DOCKER_RUN_ARGS) $(build_tag) \
		go run -mod=mod ./main.go

# Generate the static install script
gen-install: FEATURE_GATES ?= $(ALPHA_FEATURE_GATES)
gen-install: $(ensure-build-image)
	docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) bash -c \
		'helm template agones-manual --namespace agones-system $(mount_path)/install/helm/agones \
		--set agones.controller.generateTLS=false \
		--set agones.allocator.generateTLS=false \
		--set agones.allocator.generateClientTLS=false \
		--set agones.crds.cleanupOnDelete=false \
		--set agones.featureGates=$(FEATURE_GATES) \
		> $(mount_path)/install/yaml/install.yaml'

# Generate the client, conversions, deepcopy, and defaults code for our CustomResourceDefinition
gen-crd-code: $(ensure-build-image)
	docker run --rm \
		$(common_mounts) -w $(workdir_path) $(build_tag) bash -c "\
		$(git_safe) && \
		/root/gen-crd-code.sh && \
		cd $(workdir_path)/pkg && goimports -w ."

# Run a bash shell with the developer tools in it. (Creates the image if it doesn't exist)
# Can use DOCKER_RUN_ARGS for extra arguments.
shell: $(ensure-build-image)
	docker run -it --rm \
		$(common_mounts) \
		-w $(workdir_path) \
		-e "KUBECONFIG=/root/.kube/$(kubeconfig_file)" \
		$(DOCKER_RUN_ARGS) \
		$(build_tag) bash -l

# run a container with pkgsite
pkgsite:
	docker run -p 8888:8888 --rm $(common_mounts) -w $(workdir_path) \
		$(build_tag) pkgsite -http=":8888" .

# start cpu profiling pprof with a web ui
pprof-cpu-web:
	docker run --rm -it --network=host $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) \
		go tool pprof -http :6061 http://localhost:6060

# start heap profiling
pprof-heap-web:
	docker run --rm -it --network=host $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) \
		go tool pprof -http :6062 http://localhost:6060/debug/pprof/heap

# setup prometheus in the current cluster by default Persistent Volume Claims are requested.
setup-prometheus: PVC ?= true
setup-prometheus: PV_SIZE ?= 64Gi
setup-prometheus: SCRAPE_INTERVAL=30s
setup-prometheus:
	$(DOCKER_RUN) helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	$(DOCKER_RUN) helm repo update
	$(DOCKER_RUN) helm upgrade prom prometheus-community/prometheus --install --wait \
 		--namespace metrics --create-namespace \
		--set server.global.scrape_interval=$(SCRAPE_INTERVAL),server.persistentVolume.enabled=$(PVC),server.persistentVolume.size=$(PV_SIZE) \
		$(HELM_ARGS) -f $(mount_path)/build/prometheus.yaml

uninstall-prometheus: $(ensure-build-image)
	$(DOCKER_RUN) \
		helm uninstall prom --namespace=metrics

# setup grafana in the current cluster with datasource and dashboards ready for use with agones
# by default Persistent Volume Claims are requested.
setup-grafana: PVC ?= true
setup-grafana: PV_SIZE ?= 64Gi
setup-grafana: PASSWORD ?= admin
setup-grafana:
	$(DOCKER_RUN) helm repo add grafana https://grafana.github.io/helm-charts
	$(DOCKER_RUN) helm repo update
	$(DOCKER_RUN) kubectl apply -f $(mount_path)/build/grafana/
	$(DOCKER_RUN) helm upgrade grafana grafana/grafana --install --wait \
		--namespace metrics --create-namespace \
		--set persistence.enabled=$(PVC),server.persistentVolume.size=$(PV_SIZE) \
		--set adminPassword=$(PASSWORD) $(HELM_ARGS) -f $(mount_path)/build/grafana.yaml

uninstall-grafana: $(ensure-build-image)
	$(DOCKER_RUN) helm uninstall grafana --namespace=metrics
	$(DOCKER_RUN) kubectl delete -f $(mount_path)/build/grafana/

# setup prometheus-stack in the current cluster by default Persistent Volume Claims are requested.
setup-prometheus-stack: PVC ?= true
setup-prometheus-stack: PV_SIZE ?= 64Gi
setup-prometheus-stack: SCRAPE_INTERVAL=30s
setup-prometheus-stack: PASSWORD ?= admin
setup-prometheus-stack:
	$(DOCKER_RUN) helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	$(DOCKER_RUN) helm repo update
	$(DOCKER_RUN) helm upgrade prometheus-stack prometheus-community/kube-prometheus-stack --install --wait --version 19.0.2 \
 		--namespace monitoring --create-namespace \
		--set prometheus.server.global.scrape_interval=$(SCRAPE_INTERVAL) \
		--set prometheus.server.persistentVolume.enabled=$(PVC) \
		--set prometheus.server.persistentVolume.size=$(PV_SIZE) \
		--set grafana.adminPassword=$(PASSWORD) \
		$(HELM_ARGS) -f $(mount_path)/build/prometheus-stack.yaml

uninstall-prometheus-stack: $(ensure-build-image)
	$(DOCKER_RUN) \
		helm uninstall prometheus-stack --namespace=monitoring

helm-repo-update:
	$(DOCKER_RUN) helm repo update

clean-test-cluster: $(ensure-build-image) $(uninstall)

install-custom-pull-secret:
	# if IMAGE_PULL_SECRET_FILE is specified, create the agones-system namespace and install the secret
	@if [ "$(IMAGE_PULL_SECRET_FILE)" != "" ]; then \
		echo "Creating agones-system namespace..." ;\
		$(DOCKER_RUN) kubectl create namespace agones-system; \
		echo "Installing secret $(IMAGE_PULL_SECRET_FILE) in agones-system namespace..."; \
		docker run --rm $(common_mounts) -e "KUBECONFIG=/root/.kube/$(kubeconfig_file)" -v $(dir $(IMAGE_PULL_SECRET_FILE)):/root/secret $(DOCKER_RUN_ARGS) $(build_tag) \
			kubectl apply --namespace agones-system -f /root/secret/$(notdir $(IMAGE_PULL_SECRET_FILE)); \
		echo "Installing secret $(IMAGE_PULL_SECRET_FILE) in default namespace..."; \
		docker run --rm $(common_mounts) -e "KUBECONFIG=/root/.kube/$(kubeconfig_file)" -v $(dir $(IMAGE_PULL_SECRET_FILE)):/root/secret $(DOCKER_RUN_ARGS) $(build_tag) \
			kubectl apply --namespace default -f /root/secret/$(notdir $(IMAGE_PULL_SECRET_FILE)); \
	fi

update-go-deps:
	$(DOCKER_RUN) go get -u agones.dev/agones
	$(DOCKER_RUN) go mod tidy
	$(DOCKER_RUN) go mod vendor

test-gen-crd-code:
	mkdir -p build/tmp/apis | mkdir -p build/tmp/client; \
	cp -r ../pkg/apis/* build/tmp/apis | cp -r ../pkg/client/* build/tmp/client; \
	make gen-crd-code; \
	$(MAKE) diff-directory DIRECTORY=apis
	$(MAKE) diff-directory DIRECTORY=client
	# Delete build/tmp if the directory is empty
	if [ ! "$(ls -A build/tmp)" ]; then \
		echo "No differences found. Deleting empty directory build/tmp."; \
		rm -r build/tmp; \
	fi

diff-directory:
	diff_output=$$(diff -bBr build/tmp/$(DIRECTORY) ../pkg/$(DIRECTORY)); \
	if [ -z "$$diff_output" ]; then \
		echo "No differences found. Deleting build/tmp/$(DIRECTORY)"; \
		rm -r build/tmp/$(DIRECTORY); \
	else \
		echo "Differences found."; \
		echo "$$diff_output"; \
	fi

install-release: $(ensure-build-image)
	docker run -it --rm \
		$(common_mounts) \
		$(DOCKER_RUN_ARGS) \
		$(build_tag) /go/src/agones.dev/agones/build/install-release.sh

