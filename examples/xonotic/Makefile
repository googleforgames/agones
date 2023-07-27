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
# Makefile for building the Xonotic server
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

REPOSITORY = us-docker.pkg.dev/agones-images/examples

WINDOWS_VERSIONS = ltsc2019 ltsc2022
BUILDX_WINDOWS_BUILDER = windows-builder

# Build with Windows support
WITH_WINDOWS ?= 1

# When pushing set to --push
WINDOWS_DOCKER_PUSH_ARGS = --push

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
project_path := $(dir $(mkfile_path))
root_path := $(realpath $(project_path)/../..)
image_tag = $(REPOSITORY)/xonotic-example:1.2

image_tag_linux_amd64 = $(image_tag)-linux-amd64

push_server_manifest = $(image_tag_linux_amd64)
ifeq ($(WITH_WINDOWS), 1)
push_server_manifest += $(foreach windows_version, $(WINDOWS_VERSIONS), $(image_tag)-windows_amd64-$(windows_version))
endif

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

build: build-linux-image-amd64
ifeq ($(WITH_WINDOWS), 1)
build: $(foreach winver, $(WINDOWS_VERSIONS), build-windows-image-$(winver))
endif


build-linux-image-amd64:
	docker build -f $(project_path)/Dockerfile --tag=$(image_tag_linux_amd64) .

build-windows-image-%: ensure-windows-buildx
	docker buildx build --platform windows/amd64 --builder $(BUILDX_WINDOWS_BUILDER) -f $(project_path)Dockerfile.windows \
	--tag=$(image_tag)-windows_amd64-$* --build-arg WINDOWS_VERSION=$* . $(WINDOWS_DOCKER_PUSH_ARGS)

ensure-windows-buildx:
# Windows image builds must be directed to a specific buildx context.
# The default context does not support building cross platform images.
	-DOCKER_CLI_EXPERIMENTAL=enabled docker buildx create --name=$(BUILDX_WINDOWS_BUILDER) --use

# Push a docker image to the repository, and save it
push: build
	docker push $(image_tag)

# check if hosted on Google Artifact Registry
gar-check:
	gcloud container images describe $(image_tag)

#output the tag for this image
echo-image-tag:
	@echo $(image_tag)
