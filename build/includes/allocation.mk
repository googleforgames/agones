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


#         ____  ____   ____   _____           _ _	
#    __ _|  _ \|  _ \ / ___| |_   _|__   ___ | (_)_ __   __ _	
#   / _` | |_) | |_) | |       | |/ _ \ / _ \| | | '_ \ / _` |	
#  | (_| |  _ <|  __/| |___    | | (_) | (_) | | | | | | (_| |	
#   \__, |_| \_\_|    \____|   |_|\___/ \___/|_|_|_| |_|\__, |	
#   |___/                                               |___/ 

build_base_version = $(call sha,$(build_path)/build-sdk-images/tool/base/Dockerfile)
build_base_tag = agones-build-allocation-base:$(build_base_version)

# Calculate sha hash of sha hashes of all files in a specified ALLOCATION_FOLDER
allocation_build_folder = build-allocation-images/
sdk_build_folder = build-sdk-images
build_allocation_version = $(call sha_dir,$(build_path)/$(allocation_build_folder)/$(ALLOCATION_FOLDER)/*)
build_allocation_prefix = agones-build-allocation-
ALLOCATION_FOLDER ?= go
ALLOCATION_IMAGE_TAG=$(build_allocation_prefix)$(ALLOCATION_FOLDER):$(build_allocation_version)

.PHONY: gen-allocation-grpc 

# Builds the base GRPC docker image.
build-build-image-base: DOCKER_BUILD_ARGS= --build-arg GRPC_RELEASE_TAG=$(grpc_release_tag)
build-build-image-base: 
	docker build --tag=$(build_base_tag) $(build_path)$(sdk_build_folder)/tool/base $(DOCKER_BUILD_ARGS)

# create the GRPC base build image if it doesn't exist
ensure-build-image-base:
	$(MAKE) ensure-image IMAGE_TAG=$(build_base_tag) BUILD_TARGET=build-build-image-base

# create the build image allocation if it doesn't exist
ensure-build-allocation-image:
	$(MAKE) ensure-image IMAGE_TAG=$(ALLOCATION_IMAGE_TAG) BUILD_TARGET=build-build-allocation-image ALLOCATION_FOLDER=$(ALLOCATION_FOLDER)

# Builds the docker image
# Note: allocation and sdk use the same dockerfile
build-build-allocation-image: ensure-build-image-base
	docker build --tag=$(ALLOCATION_IMAGE_TAG) --build-arg BASE_IMAGE=$(build_base_tag) -f $(build_path)$(sdk_build_folder)/$(ALLOCATION_FOLDER)/Dockerfile $(build_path)$(allocation_build_folder)$(ALLOCATION_FOLDER)

# Generates grpc server and client for a single allocation, use ALLOCATION_FOLDER variable to specify the allocation folder.
gen-allocation-grpc:
	cd $(allocation_build_folder); \
	cd - ; \
	$(MAKE) ensure-build-allocation-image ALLOCATION_FOLDER=$(ALLOCATION_FOLDER) ; \
	docker run --rm $(common_mounts) -e "VERSION=$(VERSION)" $(ALLOCATION_IMAGE_TAG) gen
