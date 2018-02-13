# Copyright 2017 Google Inc. All Rights Reserved.
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
base_version = 0.1

#
# All of the following can be overwritten with environemt variables
# or passed through directly when invoking the relevent Make targets
#

# Version defaults to the short hash of the latest commit
VERSION ?= $(base_version)-$(shell git rev-parse --short HEAD)
# The registry that is being used to store docker images
REGISTRY ?= gcr.io/agones-images
# Where the kubectl configuration files are being stored
KUBEPATH ?= ~/.kube
# The (gcloud) test cluster that is being worked against
CLUSTER_NAME ?= test-cluster
# the profile to use when developing on minikube
MINIKUBE_PROFILE ?= agones

# Directory that this Makefile is in.
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
build_path := $(dir $(mkfile_path))
agones_path := $(realpath $(build_path)/..)

agones_package = agones.dev/agones
mount_path = /go/src/$(agones_package)
common_mounts = -v $(build_path)/.config/gcloud:/root/.config/gcloud \
				-v $(KUBEPATH):/root/.kube \
				-v $(agones_path):$(mount_path)

build_tag = agones-build:$(build_version)
build_remote_tag = $(REGISTRY)/$(build_tag)
controller_tag = $(REGISTRY)/agones-controller:$(VERSION)
sidecar_tag = $(REGISTRY)/agones-sdk:$(VERSION)

go_version_flags = -ldflags "-X agones.dev/agones/pkg.Version=$(VERSION)"

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

# personal includes, excluded from the git repository
-include ./local-includes/*.mk

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# build all
build: build-images build-sdks

# build the docker images
build-images: build-controller-image build-agones-sdk-image

#build all the sdks
build-sdks: build-sdk-cpp

# Run all tests
test: ensure-build-image
	docker run --rm $(common_mounts) $(build_tag) go test -race $(agones_package)/...

# Push all the images up to $(REGISTRY)
push: push-controller-image push-agones-sdk-image

# Installs the current development version of Agones into the Kubernetes cluster
install: ALWAYS_PULL_SIDECAR := true
install: IMAGE_PULL_POLICY := "Always"
install: ensure-build-image
	cp $(build_path)/install.yaml $(build_path)/.install.yaml
	sed -i -e 's!$${REGISTRY}!$(REGISTRY)!g' $(build_path)/.install.yaml
	sed -i -e 's!$${VERSION}!$(VERSION)!g' $(build_path)/.install.yaml
	sed -i -e 's!$${IMAGE_PULL_POLICY}!$(IMAGE_PULL_POLICY)!g' $(build_path)/.install.yaml
	sed -i -e 's!$${ALWAYS_PULL_SIDECAR}!$(ALWAYS_PULL_SIDECAR)!g' $(build_path)/.install.yaml
	docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) kubectl apply -f $(mount_path)/build/.install.yaml

# Build a static binary for the gameserver controller
build-controller-binary: ensure-build-image
	docker run --rm -e "CGO_ENABLED=0" $(common_mounts) $(build_tag) go build \
		-o $(mount_path)/cmd/controller/bin/controller -a $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/controller

# Build the image for the gameserver controller
build-controller-image: ensure-build-image build-controller-binary
	docker build $(agones_path)/cmd/controller/ --tag=$(controller_tag) $(DOCKER_BUILD_ARGS)

# push the gameservers controller image
push-controller-image: ensure-build-image
	docker push $(controller_tag)

# build the static binary for the gamesever sidecar
build-agones-sdk-binary: ensure-build-image
	docker run --rm -e "CGO_ENABLED=0" $(common_mounts) $(build_tag) go build \
		-o $(mount_path)/cmd/sdk-server/bin/sdk-server.linux.amd64 -a $(go_version_flags) -installsuffix cgo $(agones_package)/cmd/sdk-server
	docker run --rm -e "GOOS=darwin" -e "GOARCH=amd64" $(common_mounts) $(build_tag) go build \
		-o $(mount_path)/cmd/sdk-server/bin/sdk-server.darwin.amd64 $(go_version_flags) $(agones_package)/cmd/sdk-server
	docker run --rm -e "GOOS=windows" -e "GOARCH=amd64" $(common_mounts) $(build_tag) go build \
		-o $(mount_path)/cmd/sdk-server/bin/sdk-server.windows.amd64.exe $(go_version_flags) $(agones_package)/cmd/sdk-server

# Build the image for the gameserver sidecar
build-agones-sdk-image: ensure-build-image build-agones-sdk-binary
	docker build $(agones_path)/cmd/sdk-server/ --tag=$(sidecar_tag) $(DOCKER_BUILD_ARGS)

# Build the cpp sdk linux archive
build-sdk-cpp: ensure-build-image
	docker run --rm $(common_mounts) -w $(mount_path)/sdks/cpp $(build_tag) make build install archive VERSION=$(VERSION)

# push the gameservers sidecar image
push-agones-sdk-image: ensure-build-image
	docker push $(sidecar_tag)

# Generate the SDK gRPC server and client code
gen-gameservers-sdk-grpc: ensure-build-image
	docker run --rm $(common_mounts) -w $(mount_path) $(build_tag) /root/gen-grpc-go.sh
	docker run --rm $(common_mounts) -w $(mount_path) $(build_tag) /root/gen-grpc-cpp.sh

# Generate the client for our CustomResourceDefinition
gen-crd-client: ensure-build-image
	docker run --rm $(common_mounts) -w $(mount_path) $(build_tag) /root/gen-crd-client.sh
	docker run --rm $(common_mounts) -w $(mount_path)/pkg $(build_tag) goimports

# Run a bash shell with the developer tools in it. (Creates the image if it doesn't exist)
# Can use DOCKER_RUN_ARGS for extra arguments.
shell: ensure-build-image
	docker run -it --rm \
		$(common_mounts) \
		-w $(mount_path) \
		$(DOCKER_RUN_ARGS) \
		$(build_tag) bash -l

# run a container with godoc
godoc:
	if [ ! -f $(build_path)/.index ]; then \
		touch $(build_path)/.index && \
		docker run -p 8888:8888 --rm $(common_mounts) -v $(build_path)/.index:/root/.index \
				$(build_tag) godoc -http=":8888" -index=true -write_index=true -index_files=/root/.index;\
	fi
	docker run -p 8888:8888 --rm $(common_mounts) -v $(build_path)/.index:/root/.index \
		$(build_tag) godoc -http=":8888" -index=true -index_files=/root/.index

# Creates the build docker image
build-build-image:
	docker build --tag=$(build_tag) $(build_path)/build-image $(DOCKER_BUILD_ARGS)

# Deletes the local build docker image
clean-build-image:
	docker rmi $(build_tag)

ensure-build-config:
	-mkdir -p $(KUBEPATH)
	-mkdir -p $(build_path)/.config/gcloud

# create the build image if it doesn't exist
ensure-build-image: ensure-build-config
	@if [ -z $$(docker images -q $(build_tag)) ]; then\
		echo "Could not find $(build_tag) image. Building...";\
		$(MAKE) build-build-image;\
	fi

# attempt to pull the image, if it exists and rename it to the local tag
# exit's clean if it doesn't exist, so can be used on CI
pull-build-image:
	-docker pull $(build_remote_tag) && docker tag $(build_remote_tag) $(build_tag)

# push the local build image up to your repository
push-build-image:
	docker tag $(build_tag) $(build_remote_tag)
	docker push $(build_remote_tag)

#    ____                   _         ____ _                 _
#   / ___| ___   ___   __ _| | ___   / ___| | ___  _   _  __| |
#  | |  _ / _ \ / _ \ / _` | |/ _ \ | |   | |/ _ \| | | |/ _` |
#  | |_| | (_) | (_) | (_| | |  __/ | |___| | (_) | |_| | (_| |
#   \____|\___/ \___/ \__, |_|\___|  \____|_|\___/ \__,_|\__,_|
#                     |___/

# Initialise the gcloud login and project configuration, if you are working with GCP
gcloud-init: ensure-build-config
	docker run --rm -it $(common_mounts) $(build_tag) gcloud init

# Creates and authenticates a small, 3 node GKE cluster to work against
gcloud-test-cluster: ensure-build-image
	docker run --rm -it $(common_mounts) $(build_tag) gcloud \
		deployment-manager deployments create test-cluster --config=$(mount_path)/build/gke-test-cluster/deployment.yml
	$(MAKE) gcloud-auth-cluster

# Pulls down authentication information for kubectl against a cluster, name can be specified through CLUSTER_NAME
# (defaults to 'test-cluster')
gcloud-auth-cluster: ensure-build-image
	docker run --rm $(common_mounts) $(build_tag) gcloud config set container/cluster $(CLUSTER_NAME)
	docker run --rm $(common_mounts) $(build_tag) gcloud config set compute/zone \
		`grep zone: $(build_path)/gke-test-cluster/deployment.yml | sed 's/zone: //'`
	docker run --rm $(common_mounts) $(build_tag) gcloud container clusters get-credentials $(CLUSTER_NAME)
	-docker run --rm $(common_mounts) $(build_tag) bash -c 'kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user $$(gcloud config get-value account)'

# authenticate our docker configuration so that you can do a docker push directly
# to the gcr.io repository
gcloud-auth-docker: ensure-build-image
	-sudo rm -rf $(build_path)/tmp
	mkdir -p $(build_path)/tmp/gcloud-auth-docker
	-cp ~/.dockercfg $(build_path)/tmp/gcloud-auth-docker
	docker run --rm $(common_mounts) -v $(build_path)/tmp/gcloud-auth-docker:/root $(build_tag) gcloud docker --authorize-only
	sudo mv $(build_path)/tmp/gcloud-auth-docker/.dockercfg ~/
	sudo chown $(USER) ~/.dockercfg

# Clean the gcloud configuration
clean-gcloud-config:
	-sudo rm -r $(build_path)/.config

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
minikube-test-cluster: minikube-agones-profile
	$(MINIKUBE) start --kubernetes-version v1.8.0 --vm-driver $(MINIKUBE_DRIVER) \
		--extra-config=apiserver.Authorization.Mode=RBAC
	# wait until the master is up
	until docker run --rm $(common_mounts) --network=host -v $(minikube_cert_mount) $(DOCKER_RUN_ARGS) $(build_tag) kubectl cluster-info; \
		do \
			echo "Waiting for cluster to start..."; \
			sleep 1; \
		done
	# this is needed for kubernetes component to work correctly while RBAC is enabled
	-docker run --rm $(common_mounts) --network=host -v $(minikube_cert_mount) $(DOCKER_RUN_ARGS) $(build_tag) \
		kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default
	$(MAKE) minikube-post-start

# switch to the agones cluster
minikube-agones-profile:
	$(MINIKUBE) profile $(MINIKUBE_PROFILE)

# Connecting to minikube requires so enhanced permissions, so use this target
# instead of `make shell` to start an interactive shell for development on minikube.
minikube-shell: ensure-build-image minikube-agones-profile
	$(MAKE) shell DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount) $(DOCKER_RUN_ARGS)"

# Push the local Agones Docker images that have already been built
# via `make build` or `make build-images` into the "agones" minikube instance.
minikube-push: minikube-agones-profile
	$(MAKE) minikube-transfer-image TAG=$(sidecar_tag)
	$(MAKE) minikube-transfer-image TAG=$(controller_tag)

# Installs the current development version of Agones into the Kubernetes cluster.
# Use this instead of `make install`, as it disables PullAlways on the install.yaml
minikube-install: minikube-agones-profile
	$(MAKE) install DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)" ALWAYS_PULL_SIDECAR=false IMAGE_PULL_POLICY=IfNotPresent

# Convenience target for transferring images into minikube.
# Use TAG to specify the image to transfer into minikube
minikube-transfer-image:
	docker save $(TAG) | ($(MINIKUBE_DOCKER_ENV) && docker load)
