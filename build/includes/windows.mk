# Copyright 2018 Google Inc. All Rights Reserved.
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
# Include for Windows, operating under WSL
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

# Use a hash of the Dockerfile for the tag, so when the Dockerfile changes,
# it automatically rebuilds
build_version := $(shell sha256sum $(build_path)/build-image/Dockerfile | head -c 10)

# Minikube executable
MINIKUBE ?= minikube.exe
# Default minikube driver
MINIKUBE_DRIVER ?= hyperv
# set docker env for minikube
MINIKUBE_DOCKER_ENV ?= eval $$($(MINIKUBE) docker-env --shell=bash) && \
 export DOCKER_CERT_PATH=$$(echo $$DOCKER_CERT_PATH | $(win_to_wsl_path))

# minikube shell mount for certificates
minikube_cert_mount = $(cert_path):$(cert_path)

# transform the path from windows to WSL
win_to_wsl_path := sed -e 's|\([A-Z]\):|/\L\1|' -e 's|\\|/|g'

# find the cert path
cert_path = $(realpath $(shell $(MINIKUBE) docker-env --shell bash | grep DOCKER_CERT_PATH | awk -F "=" '{ print $$2 }' | sed 's/"//g' | $(win_to_wsl_path))/..)

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# Sets minikube credentials
minikube-post-start:
	echo "Creating minikube credentials"
	export CERT_PATH=$(cert_path) && \
	docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) kubectl config set-cluster $(MINIKUBE_PROFILE) \
		--certificate-authority=$$CERT_PATH/ca.crt --server=https://$$($(MINIKUBE) ip):8443 && \
	docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) kubectl config set-credentials $(MINIKUBE_PROFILE) \
		--client-certificate=$$CERT_PATH/client.crt --client-key=$$CERT_PATH/client.key
	docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) kubectl config set-context $(MINIKUBE_PROFILE) \
		--cluster=$(MINIKUBE_PROFILE) --user=$(MINIKUBE_PROFILE)
	docker run --rm $(common_mounts) $(DOCKER_RUN_ARGS) $(build_tag) kubectl config use-context $(MINIKUBE_PROFILE)
