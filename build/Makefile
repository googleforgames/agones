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
# Makefile for building, testing and developing Agon
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
REGISTRY ?= gcr.io/agon-images
# Where the kubectl configuration files are being stored
KUBECONFIG ?= $(build_path)/.kube
# The (gcloud) test cluster that is being worked against
CLUSTER_NAME ?= test-cluster

# Directory that this Makefile is in.
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
build_path := $(dir $(mkfile_path))
agon_path := $(realpath $(build_path)/..)

agon_package = github.com/agonio/agon
mount_path = /go/src/$(agon_package)
common_mounts = -v $(build_path)/.config/gcloud:/root/.config/gcloud \
				-v $(KUBECONFIG):/root/.kube \
				-v $(agon_path):$(mount_path)

# Use a hash of the Dockerfile for the tag, so when the Dockerfile changes,
# it automatically rebuilds
build_version := $(shell sha256sum $(build_path)/build-image/Dockerfile | head -c 10)
build_tag = agon-build:$(build_version)
controller_tag = $(REGISTRY)/gameservers-controller:$(VERSION)
sidecar_tag = $(REGISTRY)/gameservers-sidecar:$(VERSION)

go_version_flags = -ldflags "-X github.com/agonio/agon/pkg.Version=$(VERSION)"

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# build all
build: build-images build-sdks

# build the docker images
build-images: build-gameservers-controller-image build-gameservers-sidecar-image

#build all the sdks
build-sdks: build-sdk-cpp

# Run all tests
test: ensure-image
	docker run --rm $(common_mounts) --entrypoint=go $(build_tag) test -race $(agon_package)/...

# Push all the images up to $(REGISTRY)
push: push-gameservers-controller-image push-gameservers-sidecar-image

# install the development version of Agon
install:
	cp $(build_path)/install.yaml $(build_path)/.install.yaml
	sed -i -e 's!$${REGISTRY}!$(REGISTRY)!g' -e 's!$${VERSION}!$(VERSION)!g' $(build_path)/.install.yaml
	docker run --rm $(common_mounts) --entrypoint=kubectl $(build_tag) apply -f $(mount_path)/build/.install.yaml

# Build a static binary for the gameserver controller
build-gameservers-controller-binary: ensure-image
	docker run --rm -e "CGO_ENABLED=0" $(common_mounts) --entrypoint=go $(build_tag) build \
		-o $(mount_path)/gameservers/controller/bin/controller -a $(go_version_flags) -installsuffix cgo $(agon_package)/gameservers/controller

# Build the image for the gameserver controller
build-gameservers-controller-image: ensure-image build-gameservers-controller-binary
	docker build $(agon_path)/gameservers/controller/ --tag=$(controller_tag)

# push the gameservers controller image
push-gameservers-controller-image: ensure-image
	docker push $(controller_tag)

# build the static binary for the gamesever sidecar
build-gameservers-sidecar-binary: ensure-image
	docker run --rm -e "CGO_ENABLED=0" $(common_mounts) --entrypoint=go $(build_tag) build \
		-o $(mount_path)/gameservers/sidecar/bin/sidecar -a $(go_version_flags) -installsuffix cgo $(agon_package)/gameservers/sidecar

# Build the image for the gameserver sidecar
build-gameservers-sidecar-image: ensure-image build-gameservers-sidecar-binary
	docker build $(agon_path)/gameservers/sidecar/ --tag=$(sidecar_tag)

# Build the cpp sdk linux archive
build-sdk-cpp: ensure-image
	docker run --rm $(common_mounts) -w $(mount_path)/sdks/cpp --entrypoint make $(build_tag) build install archive VERSION=$(VERSION)

# push the gameservers sidecar image
push-gameservers-sidecar-image: ensure-image
	docker push $(sidecar_tag)

# Generate the SDK gRPC server and client code
gen-gameservers-sdk-grpc: ensure-image
	docker run --rm $(common_mounts) --entrypoint="/root/gen-grpc-go.sh" $(build_tag)
	docker run --rm $(common_mounts) --entrypoint="/root/gen-grpc-cpp.sh" $(build_tag)

# Generate the client for our CustomResourceDefinition
gen-crd-client: ensure-image
	docker run --rm $(common_mounts) --entrypoint="/root/gen-crd-client.sh" $(build_tag)
	docker run --rm $(common_mounts) --entrypoint=goimports $(build_tag) -w $(mount_path)/pkg

# Run a bash shell with the developer tools in it. (Creates the image if it doesn't exist)
# Can use ARGS for extra arguments.
shell: ensure-image
	docker run -it --rm \
		$(common_mounts) \
		-w $(mount_path) \
		$(ARGS) \
		--entrypoint=bash $(build_tag) -l

# run a container with godoc
godoc:
	if [ ! -f $(build_path)/.index ]; then \
		touch $(build_path)/.index && \
		docker run -p 8888:8888 --rm $(common_mounts) -v $(build_path)/.index:/root/.index \
				--entrypoint=godoc $(build_tag) -http=":8888" -index=true -write_index=true -index_files=/root/.index;\
	fi
	docker run -p 8888:8888 --rm $(common_mounts) -v $(build_path)/.index:/root/.index \
		--entrypoint=godoc $(build_tag) -http=":8888" -index=true -index_files=/root/.index

# Creates the build docker image
build-build-image:
	docker build --tag=$(build_tag) $(build_path)/build-image

# Deletes the local build docker image
clean-build-image:
	docker rmi $(build_tag)

ensure-config:
	-mkdir -p $(build_path)/.kube
	-mkdir -p $(build_path)/.config/gcloud

ensure-image: ensure-config
	@if [ -z $$(docker images -q $(build_tag)) ]; then\
		echo "Could not find $(build_tag) image. Building...";\
		$(MAKE) build-build-image;\
	fi

# Initialise the gcloud login and project configuration, if you are working with GCP
gcloud-init: ensure-config
	docker run --rm -it \
		$(common_mounts) \
		--entrypoint="gcloud" $(build_tag) init

# Creates and authenticates a small, 3 node GKE cluster to work against
gcloud-test-cluster: ensure-image
	docker run --rm -it $(common_mounts) \
		--entrypoint="gcloud" $(build_tag) \
		deployment-manager deployments create test-cluster --config=$(mount_path)/build/gke-test-cluster/deployment.yml
	$(MAKE) gcloud-auth-cluster

# Pulls down authentication information for kubectl against a cluster, name can be specified through CLUSTER_NAME
# (defaults to 'test-cluster')
gcloud-auth-cluster: ensure-image
	docker run --rm $(common_mounts) --entrypoint="gcloud" $(build_tag) config set container/cluster $(CLUSTER_NAME)
	docker run --rm $(common_mounts) --entrypoint="gcloud" $(build_tag) config set compute/zone \
		`grep zone: $(build_path)/gke-test-cluster/deployment.yml | sed 's/zone: //'`
	docker run --rm $(common_mounts) --entrypoint="gcloud" $(build_tag) container clusters get-credentials $(CLUSTER_NAME)

# authenticate our docker configuration so that you can do a docker push directly
# to the gcr.io repository
gcloud-auth-docker: ensure-image
	-sudo rm -rf /tmp/gcloud-auth-docker
	mkdir -p /tmp/gcloud-auth-docker
	-cp ~/.dockercfg /tmp/gcloud-auth-docker
	docker run --rm $(common_mounts) -v /tmp/gcloud-auth-docker:/root --entrypoint="gcloud" $(build_tag) docker --authorize-only
	sudo mv /tmp/gcloud-auth-docker/.dockercfg ~/
	sudo chown $(USER) ~/.dockercfg

# Clean the kubernetes and gcloud configuration
clean-config:
	-sudo rm -r $(build_path)/.kube
	-sudo rm -r $(build_path)/.config