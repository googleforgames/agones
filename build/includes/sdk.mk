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

# Calculate sha hash of sha hashes of all files in a specified SDK_FOLDER
build_sdk_version = $(call sha_dir,$(build_path)/build-sdk-images/$(SDK_FOLDER)/*)
build_sdk_base_remote_tag = $(REGISTRY)/$(build_sdk_base_tag)
build_sdk_prefix = agones-build-sdk-
grpc_release_tag = v1.36.1
sdk_build_folder = build-sdk-images/
examples_folder = ../examples/
SDK_FOLDER ?= go
COMMAND ?= gen
SDK_IMAGE_TAG=$(build_sdk_prefix)$(SDK_FOLDER):$(build_sdk_version)
DEFAULT_CONFORMANCE_TESTS = ready,allocate,setlabel,setannotation,gameserver,health,shutdown,watch,reserve
ALPHA_CONFORMANCE_TESTS = getplayercapacity,setplayercapacity,playerconnect,playerdisconnect,getplayercount,isplayerconnected,getconnectedplayers

.PHONY: test-sdks test-sdk build-sdks build-sdk gen-all-sdk-grpc gen-sdk-grpc run-all-sdk-command run-sdk-command build-example

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
run-all-sdk-command: run-sdk-command-go run-sdk-command-rust run-sdk-command-cpp run-sdk-command-node run-sdk-command-restapi run-sdk-command-csharp

run-sdk-command-node:
	$(MAKE) run-sdk-command COMMAND=$(COMMAND) SDK_FOLDER=node

run-sdk-command-cpp:
	$(MAKE) run-sdk-command COMMAND=$(COMMAND) SDK_FOLDER=cpp

run-sdk-command-rust:
	$(MAKE) run-sdk-command COMMAND=$(COMMAND) SDK_FOLDER=rust

run-sdk-command-go:
	$(MAKE) run-sdk-command COMMAND=$(COMMAND) SDK_FOLDER=go

run-sdk-command-restapi:
	$(MAKE) run-sdk-command COMMAND=$(COMMAND) SDK_FOLDER=restapi

run-sdk-command-csharp:
	$(MAKE) run-sdk-command COMMAND=$(COMMAND) SDK_FOLDER=csharp

# Runs a command for a specific SDK if it exists.
run-sdk-command:
	@cd $(sdk_build_folder); \
	if [ "$(SDK_FOLDER)" != "tool" ] && [ -f $(SDK_FOLDER)/$(COMMAND).sh ] ; then \
		cd - ; \
		$(MAKE) ensure-build-sdk-image SDK_FOLDER=$(SDK_FOLDER) ; \
		docker run --rm $(common_mounts) -e "VERSION=$(VERSION)" \
			$(DOCKER_RUN_ARGS) $(SDK_IMAGE_TAG) $(COMMAND) ; \
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
		docker build --tag=$(SDK_IMAGE_TAG) $(build_path)build-sdk-images/$(SDK_FOLDER) $(DOCKER_BUILD_ARGS)

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
	$(MAKE) ensure-image IMAGE_TAG=$(SDK_IMAGE_TAG) BUILD_TARGET=build-build-sdk-image SDK_FOLDER=$(SDK_FOLDER)

# Run SDK conformance Sidecar server in docker in order to run
# SDK client test against it. Useful for test development
run-sdk-conformance-local: TIMEOUT ?= 40
run-sdk-conformance-local: TESTS ?= ready,allocate,setlabel,setannotation,gameserver,health,shutdown,watch,reserve
run-sdk-conformance-local: FEATURE_GATES ?=
run-sdk-conformance-local: ensure-agones-sdk-image
	docker run -e "ADDRESS=" -p 9357:9357 -p 9358:9358 \
	 -e "TEST=$(TESTS)" -e "TIMEOUT=$(TIMEOUT)" -e "FEATURE_GATES=$(FEATURE_GATES)" $(sidecar_linux_amd64_tag)

# Run SDK conformance test, previously built, for a specific SDK_FOLDER
# Sleeps the start of the sidecar to test that the SDK blocks on connection correctly
run-sdk-conformance-no-build: TIMEOUT ?= 40
run-sdk-conformance-no-build: RANDOM := $(shell bash -c 'echo $$RANDOM')
run-sdk-conformance-no-build: DELAY ?= $(shell bash -c "echo $$[ ($(RANDOM) % 5 ) + 1 ]")
run-sdk-conformance-no-build: TESTS ?= $(DEFAULT_CONFORMANCE_TESTS)
run-sdk-conformance-no-build: GRPC_PORT ?= 9357
run-sdk-conformance-no-build: HTTP_PORT ?= 9358
run-sdk-conformance-no-build: FEATURE_GATES ?=
run-sdk-conformance-no-build: ensure-agones-sdk-image
run-sdk-conformance-no-build: ensure-build-sdk-image
	DOCKER_RUN_ARGS="--net host -e AGONES_SDK_GRPC_PORT=$(GRPC_PORT) -e AGONES_SDK_HTTP_PORT=$(HTTP_PORT) -e FEATURE_GATES=$(FEATURE_GATES) $(DOCKER_RUN_ARGS)" COMMAND=sdktest $(MAKE) run-sdk-command & \
	docker run -p $(GRPC_PORT):$(GRPC_PORT) -p $(HTTP_PORT):$(HTTP_PORT) -e "FEATURE_GATES=$(FEATURE_GATES)" -e "ADDRESS=" -e "TEST=$(TESTS)" -e "SDK_NAME=$(SDK_FOLDER)" -e "TIMEOUT=$(TIMEOUT)" -e "DELAY=$(DELAY)" \
	--net=host $(sidecar_linux_amd64_tag) --grpc-port $(GRPC_PORT) --http-port $(HTTP_PORT)

# Run SDK conformance test for a specific SDK_FOLDER
run-sdk-conformance-test: ensure-agones-sdk-image
run-sdk-conformance-test: ensure-build-sdk-image
	$(MAKE) run-sdk-command COMMAND=build-sdk-test
	$(MAKE) run-sdk-conformance-no-build

run-sdk-conformance-test-cpp:
	$(MAKE) run-sdk-conformance-test SDK_FOLDER=cpp GRPC_PORT=9003 HTTP_PORT=9103

run-sdk-conformance-test-node:
	$(MAKE) run-sdk-conformance-test SDK_FOLDER=node GRPC_PORT=9002 HTTP_PORT=9102

run-sdk-conformance-test-go:
	# run without feature flags
	$(MAKE) run-sdk-conformance-test SDK_FOLDER=go GRPC_PORT=9001 HTTP_PORT=9101
	# run with feature flags enabled
	$(MAKE) run-sdk-conformance-no-build SDK_FOLDER=go GRPC_PORT=9001 HTTP_PORT=9101 FEATURE_GATES=PlayerTracking=true TESTS=$(DEFAULT_CONFORMANCE_TESTS),$(ALPHA_CONFORMANCE_TESTS)

run-sdk-conformance-test-rust:
	# run without feature flags
	$(MAKE) run-sdk-conformance-test SDK_FOLDER=rust GRPC_PORT=9004 HTTP_PORT=9104
	# run without feature flags and with RUN_ASYNC=true
	DOCKER_RUN_ARGS="$(DOCKER_RUN_ARGS) -e RUN_ASYNC=true" $(MAKE) run-sdk-conformance-test SDK_FOLDER=rust GRPC_PORT=9004 HTTP_PORT=9104
	# run with feature flags enabled
	$(MAKE) run-sdk-conformance-test SDK_FOLDER=rust GRPC_PORT=9004 HTTP_PORT=9104 FEATURE_GATES=PlayerTracking=true TESTS=$(DEFAULT_CONFORMANCE_TESTS),$(ALPHA_CONFORMANCE_TESTS)
	# run with feature flags enabled and with RUN_ASYNC=true
	DOCKER_RUN_ARGS="$(DOCKER_RUN_ARGS) -e RUN_ASYNC=true" $(MAKE) run-sdk-conformance-test SDK_FOLDER=rust GRPC_PORT=9004 HTTP_PORT=9104 FEATURE_GATES=PlayerTracking=true TESTS=$(DEFAULT_CONFORMANCE_TESTS),$(ALPHA_CONFORMANCE_TESTS)

run-sdk-conformance-test-rest:
	# run without feature flags
	$(MAKE) run-sdk-conformance-test SDK_FOLDER=restapi HTTP_PORT=9050
	# run with feature flags enabled
	$(MAKE) run-sdk-conformance-no-build SDK_FOLDER=restapi GRPC_PORT=9001 HTTP_PORT=9101 FEATURE_GATES=PlayerTracking=true TESTS=$(DEFAULT_CONFORMANCE_TESTS),$(ALPHA_CONFORMANCE_TESTS)

	$(MAKE) run-sdk-command COMMAND=clean SDK_FOLDER=restapi

# Run a conformance test for all SDKs supported
run-sdk-conformance-tests: run-sdk-conformance-test-node run-sdk-conformance-test-go run-sdk-conformance-test-rust run-sdk-conformance-test-rest run-sdk-conformance-test-cpp

# Clean package directories and binary files left
# after building conformance tests for all SDKs supported
clean-sdk-conformance-tests:
	$(MAKE) run-all-sdk-command COMMAND=clean

# Start a shell in the SDK image. This is primarily used for publishing packages.
# Using a shell is the easiest, because of Google internal processes and interactive commands required.
sdk-shell:
	$(MAKE) ensure-build-sdk-image SDK_FOLDER=$(SDK_FOLDER)
	docker run --rm -it $(common_mounts) -v ~/.ssh:/tmp/.ssh:ro $(DOCKER_RUN_ARGS) \
 	--entrypoint=/root/shell.sh $(SDK_IMAGE_TAG)

# SDK shell for node
sdk-shell-node:
	$(MAKE) sdk-shell SDK_FOLDER=node

# SDK shell for csharp
sdk-shell-csharp:
	$(MAKE) sdk-shell SDK_FOLDER=csharp

# Publish csharp SDK to NuGet
sdk-publish-csharp: RELEASE_VERSION ?= $(base_version)
sdk-publish-csharp:
	$(MAKE) run-sdk-command-csharp COMMAND=publish VERSION=$(RELEASE_VERSION) DOCKER_RUN_ARGS="$(DOCKER_RUN_ARGS) -it"

# Perform make build for all examples
build-examples: build-example-xonotic build-example-cpp-simple build-example-autoscaler-webhook build-example-nodejs-simple

# Run "make build" command for one example directory
build-example:
	cd  $(examples_folder)/$(EXAMPLE); \
	if [ -f Makefile ] ; then \
		make build; \
	else \
		echo "Makefile was not found in "/examples/$(EXAMPLE)" directory - nothing to execute" ; \
	fi

build-example-xonotic:
	$(MAKE) build-example EXAMPLE=xonotic

build-example-cpp-simple:
	$(MAKE) build-example EXAMPLE=cpp-simple

build-example-rust-simple:
	$(MAKE) build-example EXAMPLE=rust-simple

build-example-autoscaler-webhook:
	$(MAKE) build-example EXAMPLE=autoscaler-webhook

build-example-nodejs-simple:
	$(MAKE) build-example EXAMPLE=nodejs-simple
