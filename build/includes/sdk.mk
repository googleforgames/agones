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

build_sdk_base_version = $(call sha,$(build_path)/build-sdk-images/tool/base/Dockerfile)
build_sdk_base_tag = agones-build-sdk-base:$(build_sdk_base_version)
build_sdk_base_remote_tag = $(REGISTRY)/$(build_sdk_base_tag)
build_sdk_prefix = agones-build-sdk-
grpc_release_tag = v1.16.1
sdk_build_folder = build-sdk-images/
SDK_FOLDER ?= go
COMMAND ?= gen

.PHONY: test-sdks test-sdk build-sdks build-sdk gen-all-sdk-grpc gen-sdk-grpc run-all-sdk-command run-sdk-command 

# Tests all the sdks
test-sdks: COMMAND := test
test-sdks: run-all-sdk-command

# Tests a single sdk, use SDK_FOLDER variable to specify the sdk folder.
test-sdk: COMMAND := test
test-sdk: run-sdk-command 

# Builds all the sdks
build-sdks: COMMAND := build
build-sdks: run-all-sdk-command

# Builds a single sdk, use SDK_FOLDER variable to specify the sdk folder.
build-sdk: COMMAND := build
build-sdk: run-sdk-command

# Generates grpc server and client for all supported languages.
gen-all-sdk-grpc: COMMAND := gen
gen-all-sdk-grpc: run-all-sdk-command

# Generates grpc server and client for a single sdk, use SDK_FOLDER variable to specify the sdk folder.
gen-sdk-grpc: COMMAND := gen
gen-sdk-grpc: run-sdk-command

# Runs a command on all supported languages, use COMMAND variable to select which command.
run-all-sdk-command: run-sdk-command-go run-sdk-command-rust run-sdk-command-cpp run-sdk-command-node

run-sdk-command-node:
	$(MAKE) run-sdk-command COMMAND=$(COMMAND) SDK_FOLDER=node

run-sdk-command-cpp:
	$(MAKE) run-sdk-command COMMAND=$(COMMAND) SDK_FOLDER=cpp

run-sdk-command-rust:
	$(MAKE) run-sdk-command COMMAND=$(COMMAND) SDK_FOLDER=rust

run-sdk-command-go:
	$(MAKE) run-sdk-command COMMAND=$(COMMAND) SDK_FOLDER=go

# Runs a command for a specific SDK if it exists.
run-sdk-command:
	cd $(sdk_build_folder); \
	if [ "$(SDK_FOLDER)" != "tool" ] && [ -f $(SDK_FOLDER)/$(COMMAND).sh ] ; then \
		cd - ; \
		$(MAKE) ensure-build-sdk-image SDK_FOLDER=$(SDK_FOLDER) ; \
		docker run --rm $(common_mounts) -e "VERSION=$(VERSION)" \
			$(DOCKER_RUN_ARGS) $(build_sdk_prefix)$(SDK_FOLDER):$(build_version) $(COMMAND) ; \
	else \
		echo "Command $(COMMAND) not found - nothing to execute" ; \
	fi

# Builds the base GRPC docker image.
build-build-sdk-image-base: DOCKER_BUILD_ARGS= --build-arg GRPC_RELEASE_TAG=$(grpc_release_tag)
build-build-sdk-image-base: 
	docker build --tag=$(build_sdk_base_tag) $(build_path)build-sdk-images/tool/base $(DOCKER_BUILD_ARGS)

# Builds the docker image used by commands for a specific sdk
build-build-sdk-image: DOCKER_BUILD_ARGS= --build-arg BASE_IMAGE=$(build_sdk_base_tag)
build-build-sdk-image: ensure-build-sdk-image-base
		docker build --tag=$(build_sdk_prefix)$(SDK_FOLDER):$(build_version) $(build_path)build-sdk-images/$(SDK_FOLDER) $(DOCKER_BUILD_ARGS)

# attempt to pull the image, if it exists and rename it to the local tag
# exit's clean if it doesn't exist, so can be used on CI
pull-build-sdk-base-image:
	$(MAKE) pull-remote-build-image REMOTE_TAG=$(build_sdk_base_remote_tag) LOCAL_TAG=$(build_sdk_base_tag)

# push the local build image up to your repository
push-build-sdk-base-image:
	$(MAKE) push-remote-build-image REMOTE_TAG=$(build_sdk_base_remote_tag) LOCAL_TAG=$(build_sdk_base_tag)

# create the sdk base build image if it doesn't exist
ensure-build-sdk-image-base:
	$(MAKE) ensure-image IMAGE_TAG=$(build_sdk_base_tag) BUILD_TARGET=build-build-sdk-image-base

# create the build image sdk if it doesn't exist
ensure-build-sdk-image:
	$(MAKE) build-build-sdk-image SDK_FOLDER=$(SDK_FOLDER)

# Run SDK conformance Sidecar server in docker in order to run
# SDK client test against it. Useful for test development
SECONDS ?= 30
run-sdk-conformance-local:
	docker run -e "ADDRESS=" -p 59357:59357 \
	 -e "TEST=ready,allocate,setlabel,setannotation,gameserver,health,shutdown,watch" -e "TIMEOUT=$(SECONDS)" $(sidecar_tag)

# Run SDK conformance test for a specific SDK_FOLDER
SECONDS ?= 30
run-sdk-conformance-test: SDK_FOLDER=$(SDK_FOLDER)
run-sdk-conformance-test: ensure-build-sdk-image
	sleep 2s && DOCKER_RUN_ARGS="--network=host $(DOCKER_RUN_ARGS)" COMMAND=sdktest $(MAKE) run-sdk-command & \
	docker run -p 59357:59357 -e "ADDRESS=" \
	 -e "TEST=ready,allocate,setlabel,setannotation,gameserver,health,shutdown,watch" -e "TIMEOUT=$(SECONDS)" --net=host  $(sidecar_tag)

# Run a conformance test for all SDKs supported
run-sdk-conformance-tests: ensure-agones-sdk-image
	$(MAKE) run-sdk-command SDK_FOLDER=node COMMAND=build
	$(MAKE) run-sdk-conformance-test SDK_FOLDER=node
	$(MAKE) run-sdk-conformance-test SDK_FOLDER=go
	$(MAKE) run-sdk-command SDK_FOLDER=node COMMAND=clean
