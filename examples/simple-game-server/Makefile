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
# Makefile for building a simple game-server
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

REGISTRY = us-docker.pkg.dev/agones-images/examples
BUILDX_WINDOWS_BUILDER = windows-builder
# Versions of Windows Server to support, default is Windows Server 2019.
# For the full list of tags see https://hub.docker.com/_/microsoft-windows-servercore.
# GKE-Windows version map: https://cloud.google.com/kubernetes-engine/docs/how-to/creating-a-cluster-windows#version_mapping
WINDOWS_VERSIONS = ltsc2019
WINDOWS_DOCKER_PUSH_ARGS = # When pushing set to --push.
# Build with Windows support
WITH_WINDOWS=1
# Build with ARM64 support
WITH_ARM64=1

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
project_path := $(dir $(mkfile_path))
server_tag = $(REGISTRY)/simple-game-server:0.14

ifeq ($(WITH_WINDOWS), 1)
server_tag_linux_amd64 = $(server_tag)-linux-amd64
server_tag_linux_arm64 = $(server_tag)-linux-arm64
else
server_tag_linux_amd64 = $(server_tag)-amd64
server_tag_linux_arm64 = $(server_tag)-arm64
endif

push_server_manifest = $(server_tag_linux_amd64)
root_path = $(realpath $(project_path)/../..)

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

ifeq ($(WITH_WINDOWS), 1)
push_server_manifest += $(foreach windows_version, $(WINDOWS_VERSIONS), $(server_tag)-windows_amd64-$(windows_version))
endif
ifeq ($(WITH_ARM64), 1)
push_server_manifest += $(server_tag_linux_arm64)
endif

build: build-linux-image-amd64

ifeq ($(WITH_WINDOWS), 1)
build: $(foreach winver, $(WINDOWS_VERSIONS), build-windows-image-$(winver))
endif
ifeq ($(WITH_ARM64), 1)
build: build-linux-image-arm64
endif

# Builds all image artifacts and create a docker manifest that is used to inform the CRI (Docker or containerd usually)
# which image is the best fit for the host. See https://www.docker.com/blog/multi-arch-images/ for details.
push: push-linux-image-amd64
ifeq ($(WITH_WINDOWS), 1)
push: push-windows-images
endif
ifeq ($(WITH_ARM64), 1)
push: push-linux-image-arm64
endif
	-DOCKER_CLI_EXPERIMENTAL=enabled docker manifest rm $(server_tag)
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create $(server_tag) $(push_server_manifest)
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push $(server_tag) --purge

# Pushes all variants of the Windows images to the container image registry.
push-linux-image-amd64: build
	docker push $(server_tag_linux_amd64)
push-linux-image-arm64: build
	$(MAKE) DOCKER_BUILD_ARGS=--push build-linux-image-arm64

# Pushes all variants of the Windows images to the container image registry.
push-windows-images: $(foreach winver, $(WINDOWS_VERSIONS), push-windows-image-$(winver))

# Pushes a specific version of Windows to a container image registry.
push-windows-image-%:
	$(MAKE) WINDOWS_DOCKER_PUSH_ARGS=--push build-windows-image-$*

build-windows-image-%: build-windows-binary ensure-windows-buildx
	cd $(root_path) && DOCKER_CLI_EXPERIMENTAL=enabled \
		docker buildx build --platform windows/amd64 --builder $(BUILDX_WINDOWS_BUILDER) -f $(project_path)Dockerfile.windows \
	 	--tag=$(server_tag)-windows_amd64-$* --build-arg WINDOWS_VERSION=$* examples/simple-game-server/ $(WINDOWS_DOCKER_PUSH_ARGS)

# Builds the server binary for Windows (amd64).
build-windows-binary:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/simple-game-server.exe main.go

ensure-windows-buildx:
	# TODO: Remove `--use` and specify `--builder` for each buildx command once Docker 19.03.13 is available.
	# `--use` sets the global default buildx context to $(BUILDX_WINDOWS_BUILDER).
	#
	# Windows image builds must be directed to a specific buildx context.
	# The default context does not support building cross platform images.
	-DOCKER_CLI_EXPERIMENTAL=enabled docker buildx create --name=$(BUILDX_WINDOWS_BUILDER) --use

# Build a docker image for the server, and tag it
build-linux-image-amd64:
	cd $(root_path) && docker build -f $(project_path)Dockerfile --tag=$(server_tag_linux_amd64) .
build-linux-image-arm64:
	cd $(root_path) && docker buildx build --platform linux/arm64 -f $(project_path)Dockerfile $(DOCKER_BUILD_ARGS) --tag=$(server_tag_linux_arm64) .

# check if hosted on Google Artifact Registry
gar-check:
	gcloud container images describe $(server_tag)

#output the tag for this image
echo-image-tag:
	@echo $(server_tag)
