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
base_version = 1.16.0

# agones image release registry
release_registry = gcr.io/agones-images

#
# All of the following can be overwritten with environment variables
# or passed through directly when invoking the relevent Make targets
#

# Version defaults to the short hash of the latest commit
VERSION ?= $(base_version)-$(shell git rev-parse --short=7 HEAD)
# The registry that is being used to store docker images
REGISTRY ?= $(release_registry)
# kubectl configuration to use
KUBECONFIG ?= ~/.kube/config
# The (gcloud) test cluster that is being worked against
GCP_CLUSTER_NAME ?= test-cluster
GCP_CLUSTER_ZONE ?= us-west1-c
GCP_BUCKET_CHARTS ?= agones-chart
# the profile to use when developing on minikube
MINIKUBE_PROFILE ?= agones
GO_BUILD_TAGS ?= none
BUILDX_WINDOWS_BUILDER = windows-builder
WINDOWS_DOCKER_PUSH_ARGS =
# Versions of Windows Server to support, default is Windows Server 2019.
# For the full list of tags see https://hub.docker.com/_/microsoft-windows-servercore.
# GKE-Windows version map: https://cloud.google.com/kubernetes-engine/docs/how-to/creating-a-cluster-windows#version_mapping
WINDOWS_VERSIONS = ltsc2019

# Specify stress test level 1..100
# STRESS_TEST_LEVEL=n requires capacity between 50*n up to 100*n simple-game-server Game Servers.
STRESS_TEST_LEVEL ?= 20

# kind cluster name to use
KIND_PROFILE ?= agones
KIND_CONTAINER_NAME=$(KIND_PROFILE)-control-plane

# Game Server image to use while doing end-to-end tests
GS_TEST_IMAGE ?= gcr.io/agones-images/simple-game-server:0.3

ALPHA_FEATURE_GATES ?= "PlayerTracking=true&NodeExternalDNS=true&CustomFasSyncInterval=true"

# Build with Windows support
WITH_WINDOWS=1

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
workdir_path = $(mount_path)

build_version = $(call sha,$(build_path)/build-image/Dockerfile)
build_tag = agones-build:$(build_version)
controller_tag = $(REGISTRY)/agones-controller:$(VERSION)
sidecar_tag = $(REGISTRY)/agones-sdk:$(VERSION)

ifeq ($(WITH_WINDOWS), 1)
sidecar_linux_amd64_tag = $(sidecar_tag)-linux_amd64
else
sidecar_linux_amd64_tag = $(sidecar_tag)
endif

ping_tag = $(REGISTRY)/agones-ping:$(VERSION)
allocator_tag = $(REGISTRY)/agones-allocator:$(VERSION)

gomod_on = GO111MODULE=on

go_version_flags = -ldflags "-X agones.dev/agones/pkg.Version=$(VERSION)"
DOCKER_RUN ?= docker run --rm $(common_mounts) -e "KUBECONFIG=/root/.kube/$(kubeconfig_file)" -e "$(gomod_on)" -w $(workdir_path) $(DOCKER_RUN_ARGS) $(build_tag)


ifdef DOCKER_RUN
	ensure-build-image += ensure-build-image
endif

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

go_build_cmd = go build -mod=vendor
GO_BUILD_LINUX_AMD64=docker run --rm -e "CGO_ENABLED=0" -e "$(gomod_on)" -w $(workdir_path) $(common_mounts) $(build_tag) $(go_build_cmd)
GO_BUILD_DARWIN_AMD64=docker run --rm -e "GOOS=darwin" -e "GOARCH=amd64" -e "$(gomod_on)" -w $(workdir_path) $(common_mounts) $(build_tag) $(go_build_cmd)
GO_BUILD_WINDOWS_AMD64=docker run --rm -e "GOOS=windows" -e "GOARCH=amd64" -e "$(gomod_on)" -w $(workdir_path) $(common_mounts) $(build_tag) $(go_build_cmd)

RACE_DETECTOR_ARGS=-race

ifdef NO_RACE_DETECTOR
  RACE_DETECTOR_ARGS=
endif

go_test_cmd = go test -mod=vendor $(RACE_DETECTOR_ARGS)
GO_TEST=$(DOCKER_RUN) $(go_test_cmd)
GO_E2E_TEST_ARGS=--kubeconfig /root/.kube/$(kubeconfig_file)

PERF_OUTPUT_DIR=$(mount_path)/build/.perf

go_build_base_path=$(mount_path)

ifdef LOCAL_GO
	# use local Go tooling, which greatly speeds up incremental rebuilds, in particular on macOS
	# Note that agones.dev must be in GOPATH
	GO_BUILD_LINUX_AMD64=CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(gomod_on) $(go_build_cmd)
	GO_BUILD_WINDOWS_AMD64=GOOS=windows GOARCH=amd64 $(gomod_on) $(go_build_cmd)
	GO_BUILD_DARWIN_AMD64=GOOS=darwin GOARCH=amd64 $(gomod_on) $(go_build_cmd)
	GO_TEST=$(go_test_cmd) -v
	GO_E2E_TEST_ARGS=
	PERF_OUTPUT_DIR=$(build_path)/.perf
	go_build_base_path=$(agones_path)
endif

ZIP_SDK=docker run --rm $(common_mounts) -w $(workdir_path)/cmd/sdk-server/bin/ $(build_tag) zip $(zip_flags)
ifdef LOCAL_ZIP
	ZIP_SDK=cd $(agones_path)/cmd/sdk-server/bin && zip -0
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

# build all
build: build-images build-sdks

# build the docker images
build-images: build-controller-image build-agones-sdk-image build-ping-image build-allocator-image
ifeq ($(WITH_WINDOWS), 1)
build-images: build-agones-sdk-image-windows
endif

# package the current agones helm chart
build-chart: RELEASE_VERSION ?= $(base_version)
build-chart: CHART_DIR ?= install/helm/agones/
build-chart: $(ensure-build-image)
	docker run --rm $(common_mounts) -w $(workdir_path) $(build_tag) bash -c \
		"mkdir -p install/helm/bin/ && rm -rf install/helm/bin/* && \
			helm package -d install/helm/bin/ --version $(RELEASE_VERSION) $(CHART_DIR)"

# push the current chart to google cloud storage and update the index
push-chart: $(ensure-build-image) build-chart
	docker run --rm $(common_mounts) -w $(workdir_path) $(build_tag) bash -c \
		"gsutil copy gs://$(GCP_BUCKET_CHARTS)/index.yaml ./install/helm/bin/index.yaml || /bin/true && \
		helm repo index --merge ./install/helm/bin/index.yaml ./install/helm/bin && \
		cat ./install/helm/bin/index.yaml && ls ./install/helm/bin/ && \
		cp ./install/helm/bin/index.yaml ./install/helm/bin/index-$(VERSION).yaml && \
		gsutil copy ./install/helm/bin/*.* gs://$(GCP_BUCKET_CHARTS)/"

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
	$(MAKE) test-e2e-integration
	$(MAKE) test-e2e-failure


# If GAMESERVERS_NAMESPACE is empty - random namespace will be created
test-e2e-integration: FEATURE_GATES ?= $(ALPHA_FEATURE_GATES)
test-e2e-integration: GAMESERVERS_NAMESPACE ?= ""
test-e2e-integration: $(ensure-build-image)
	echo "Starting e2e integration test!"
	$(GO_TEST) $(ARGS) $(agones_package)/test/e2e $(GO_E2E_TEST_ARGS) \
		--gameserver-image=$(GS_TEST_IMAGE) \
		--pullsecret=$(IMAGE_PULL_SECRET) \
		--feature-gates=$(FEATURE_GATES) \
		--namespace=$(GAMESERVERS_NAMESPACE)
	echo "Finishing e2e integration test!"

test-e2e-failure: $(ensure-build-image)
	echo "starting e2e controller failure test"
	$(GO_TEST) -parallel=1 $(agones_package)/test/e2e/controller $(GO_E2E_TEST_ARGS) \
    	--gameserver-image=$(GS_TEST_IMAGE) \
    	--pullsecret=$(IMAGE_PULL_SECRET)
	echo "Finishing e2e controller failure test!"

# Runs end-to-end stress tests on the current configured cluster
# For minikube user the minikube-stress-test-e2e targets
stress-test-e2e: $(ensure-build-image)
	$(GO_TEST) $(agones_package)/test/e2e $(ARGS) $(GO_E2E_TEST_ARGS) \
		-timeout 1h \
		-run '.*StressTest.*' \
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
push: push-controller-image push-agones-sdk-image push-ping-image push-allocator-image

# Installs the current development version of Agones into the Kubernetes cluster
install: ALWAYS_PULL_SIDECAR := true
install: IMAGE_PULL_POLICY := "Always"
install: PING_SERVICE_TYPE := "LoadBalancer"
install: ALLOCATOR_SERVICE_TYPE := "LoadBalancer"
install: CRD_CLEANUP := true
install: LOG_LEVEL := "debug"
install: EXTERNAL_IP ?= $(shell $(DOCKER_RUN) kubectl get services agones-allocator -n agones-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
install: FEATURE_GATES ?= $(ALPHA_FEATURE_GATES)
install: HELM_ARGS ?=
install: $(ensure-build-image) install-custom-pull-secret
	$(DOCKER_RUN) \
		helm upgrade --install --atomic --namespace=agones-system \
		--create-namespace \
		--set agones.image.tag=$(VERSION),agones.image.registry=$(REGISTRY) \
		--set agones.image.controller.pullPolicy=$(IMAGE_PULL_POLICY),agones.image.sdk.alwaysPull=$(ALWAYS_PULL_SIDECAR) \
		--set agones.image.controller.pullSecret=$(IMAGE_PULL_SECRET) \
		--set agones.ping.http.serviceType=$(PING_SERVICE_TYPE),agones.ping.udp.serviceType=$(PING_SERVICE_TYPE) \
		--set agones.allocator.http.serviceType=$(ALLOCATOR_SERVICE_TYPE) \
		--set agones.controller.logLevel=$(LOG_LEVEL) \
		--set agones.crds.cleanupOnDelete=$(CRD_CLEANUP) \
		--set agones.featureGates=$(FEATURE_GATES) \
		--set agones.allocator.http.loadBalancerIP=$(EXTERNAL_IP) \
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
build-controller-binary: $(ensure-build-image)
	$(GO_BUILD_LINUX_AMD64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/controller/bin/controller \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/controller

# Lint the go source code.
# use LINT_TIMEOUT to manipulate the linter timeout
lint: LINT_TIMEOUT ?= 15m
lint: $(ensure-build-image)
	docker run -t -e "TERM=xterm-256color" -e "$(gomod_on)" --rm $(common_mounts) -w $(workdir_path) $(DOCKER_RUN_ARGS) $(build_tag) bash -c \
		"golangci-lint run ./examples/... && golangci-lint run --timeout $(LINT_TIMEOUT) ./..."

# Extract licenses from source tree
build-licenses:
	docker run --rm $(common_mounts) $(build_tag) $(mount_path)/build/extract-licenses.sh

# Tarball source for dependencies that are required to be distributed with the image (MPL)
build-required-src-dist:
	docker run --rm $(common_mounts) $(build_tag) $(mount_path)/build/build-required-src-dist.sh

# Build the image for the gameserver controller
build-controller-image: $(ensure-build-image) build-controller-binary build-licenses build-required-src-dist
	docker build $(agones_path)/cmd/controller/ --tag=$(controller_tag) $(DOCKER_BUILD_ARGS)

# push the gameservers controller image
push-controller-image: $(ensure-build-image)
	docker push $(controller_tag)

# build the static binary for the gameserver sidecar
build-agones-sdk-binary: $(ensure-build-image) build-agones-sdk-binary-linux build-agones-sdk-binary-windows build-agones-sdk-binary-darwin
	$(ZIP_SDK) \
		agonessdk-server-$(VERSION).zip sdk-server.darwin.amd64 sdk-server.linux.amd64 sdk-server.windows.amd64.exe

# build the static binary for the gameserver sidecar for Linux
build-agones-sdk-binary-linux: $(ensure-build-image)
	$(GO_BUILD_LINUX_AMD64) \
		-o $(go_build_base_path)/cmd/sdk-server/bin/sdk-server.linux.amd64 $(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/sdk-server

# build the static binary for the gameserver sidecar for Darwin (macOS)
build-agones-sdk-binary-darwin: $(ensure-build-image)
	$(GO_BUILD_DARWIN_AMD64) \
		-o $(go_build_base_path)/cmd/sdk-server/bin/sdk-server.darwin.amd64 $(go_rebuild_flags) $(go_version_flags) $(agones_package)/cmd/sdk-server

# build the windows binary for the gameserver sidecar for Windows
build-agones-sdk-binary-windows: $(ensure-build-image)
	$(GO_BUILD_WINDOWS_AMD64) \
		-o $(go_build_base_path)/cmd/sdk-server/bin/sdk-server.windows.amd64.exe $(go_rebuild_flags) $(go_version_flags) $(agones_package)/cmd/sdk-server

ensure-windows-buildx:
	# TODO: Remove `--use` and specify `--builder` for each buildx command once Docker 19.03.13 is available.
	# `--use` sets the global default buildx context to $(BUILDX_WINDOWS_BUILDER).
	#
	# Windows image builds must be directed to a specific buildx context.
	# The default context does not support building cross platform images.
	-DOCKER_CLI_EXPERIMENTAL=enabled docker buildx create --name=$(BUILDX_WINDOWS_BUILDER) --use

# Build the image for the gameserver sidecar and SDK binaries
build-agones-sdk-image: $(ensure-build-image) build-agones-sdk-binary build-licenses build-required-src-dist
	docker build $(agones_path)/cmd/sdk-server/ --tag=$(sidecar_linux_amd64_tag) $(DOCKER_BUILD_ARGS)

build-agones-sdk-image-windows: $(foreach winver, $(WINDOWS_VERSIONS), build-agones-sdk-image-windows-$(winver))

# Build the image for the gameserver sidecar and SDK binaries
build-agones-sdk-image-windows-%: $(ensure-build-image) build-agones-sdk-binary build-licenses build-required-src-dist ensure-windows-buildx
	# TODO: Use `--builder $(BUILDX_WINDOWS_BUILDER)` later once Docker 19.03.13 is the build target. Cloud Build has 19.03.08 https://cloud.google.com/cloud-build/release-notes
	DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform windows/amd64 -f $(agones_path)/cmd/sdk-server/Dockerfile.windows --tag=$(sidecar_tag)-windows_amd64-$* --build-arg WINDOWS_VERSION=$* $(DOCKER_BUILD_ARGS) $(agones_path)/cmd/sdk-server/ $(WINDOWS_DOCKER_PUSH_ARGS)

# Build sidecar image only
build-agones-sdk-server-image: $(ensure-build-image) build-agones-sdk-binary-linux build-licenses build-required-src-dist
	docker build $(agones_path)/cmd/sdk-server/ --tag=$(sidecar_linux_amd64_tag) $(DOCKER_BUILD_ARGS)

build-agones-sdk-server-image-windows: $(foreach winver, $(WINDOWS_VERSIONS), build-agones-sdk-server-image-windows-$(winver))

# Build sidecar image only
build-agones-sdk-server-image-windows-%: $(ensure-build-image) build-agones-sdk-binary-linux build-licenses build-required-src-dist ensure-windows-buildx
	# TODO: Use `--builder $(BUILDX_WINDOWS_BUILDER)` later once Docker 19.03.13 is the build target. Cloud Build has 19.03.08 https://cloud.google.com/cloud-build/release-notes
	DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build $(agones_path)/cmd/sdk-server/ -f $(agones_path)/cmd/sdk-server/Dockerfile.windows --platform windows/amd64 --tag=$(sidecar_tag)-windows_amd64-$* --build-arg WINDOWS_VERSION=$* $(DOCKER_BUILD_ARGS) $(WINDOWS_DOCKER_PUSH_ARGS)

# Build a static binary for the ping service
build-ping-binary: $(ensure-build-image)
	$(GO_BUILD_LINUX_AMD64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/ping/bin/ping \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/ping

# Pushes up the ping image
push-ping-image: $(ensure-build-image)
	docker push $(ping_tag)

# Build the image for the ping service
build-ping-image: $(ensure-build-image) build-ping-binary build-licenses build-required-src-dist
	docker build $(agones_path)/cmd/ping/ --tag=$(ping_tag) $(DOCKER_BUILD_ARGS)

# Build a static binary for the allocator service
build-allocator-binary: $(ensure-build-image)
	$(GO_BUILD_LINUX_AMD64) \
	-tags $(GO_BUILD_TAGS) -o $(go_build_base_path)/cmd/allocator/bin/allocator \
	$(go_rebuild_flags) $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/allocator

# Pushes up the allocator image
push-allocator-image: $(ensure-build-image)
	docker push $(allocator_tag)

# Build the image for the allocator service
build-allocator-image: $(ensure-build-image) build-allocator-binary build-licenses build-required-src-dist
	docker build $(agones_path)/cmd/allocator/ --tag=$(allocator_tag) $(DOCKER_BUILD_ARGS)

ifeq ($(WITH_WINDOWS), 1)
# push the gameservers sidecar image
push-agones-sdk-image: push-agones-sdk-linux-image
	# Ensure that the sidecar manifest is removed before creating a new one.
	-DOCKER_CLI_EXPERIMENTAL=enabled docker manifest rm $(sidecar_tag)
	# Using docker buildx with foreign OSes is a bit awkward. Since the layers but not the tag
	# was cached we have to rebuild the image (which should be fast since it's cached).
	$(MAKE) WINDOWS_DOCKER_PUSH_ARGS=--push build-agones-sdk-image-windows
	# Agones should start using the -linux_amd64 suffix for Linux images. This allows multi-arch support in the future.
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create $(sidecar_tag) $(sidecar_linux_amd64_tag) $(foreach windows_version, $(WINDOWS_VERSIONS), $(sidecar_tag)-windows_amd64-$(windows_version))
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push $(sidecar_tag)
else
push-agones-sdk-image: push-agones-sdk-linux-image
endif

# push the gameservers sidecar image
push-agones-sdk-linux-image: $(ensure-build-image)
	docker push $(sidecar_linux_amd64_tag)

# Generate the Embedded CRD openapi
gen-embedded-openapi:
	docker run --rm $(common_mounts) -w $(workdir_path)/build $(build_tag) ./export-openapi.sh

# Generate the static install script
gen-install: $(ensure-build-image)
	docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) bash -c \
		'helm template agones-manual --namespace agones-system $(mount_path)/install/helm/agones \
		--set agones.controller.generateTLS=false \
		--set agones.allocator.generateTLS=false \
		--set agones.allocator.generateClientTLS=false \
		--set agones.crds.cleanupOnDelete=false \
		> $(mount_path)/install/yaml/install.yaml'

# Generate the client for our CustomResourceDefinition
gen-crd-client: $(ensure-build-image)
	docker run --rm $(common_mounts) -w $(workdir_path) $(build_tag) /root/gen-crd-client.sh
	docker run --rm $(common_mounts) -w $(workdir_path)/pkg $(build_tag) goimports -w .

# Run a bash shell with the developer tools in it. (Creates the image if it doesn't exist)
# Can use DOCKER_RUN_ARGS for extra arguments.
shell: $(ensure-build-image)
	docker run -it --rm \
		$(common_mounts) \
		-w $(workdir_path) \
		-e "KUBECONFIG=/root/.kube/$(kubeconfig_file)" \
		$(DOCKER_RUN_ARGS) \
		$(build_tag) bash -l

# run a container with godoc
godoc:
	docker run -p 8888:8888 --rm $(common_mounts) -v $(build_path)/.index:/root/.index \
		$(build_tag) godoc -http=":8888" -index=true

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
	$(DOCKER_RUN) helm upgrade prom prometheus-community/prometheus --install --wait --version 11.16.2 \
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
	$(DOCKER_RUN) helm upgrade grafana grafana/grafana --install --wait --version=5.7.10 \
		--namespace metrics --create-namespace \
		--set persistence.enabled=$(PVC),server.persistentVolume.size=$(PV_SIZE) \
		--set adminPassword=$(PASSWORD) $(HELM_ARGS) -f $(mount_path)/build/grafana.yaml

uninstall-grafana: $(ensure-build-image)
	$(DOCKER_RUN) helm uninstall grafana --namespace=metrics
	$(DOCKER_RUN) kubectl delete -f $(mount_path)/build/grafana/

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
