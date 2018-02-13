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
# Include for OSX operating System
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

# Use a hash of the Dockerfile for the tag, so when the Dockerfile changes,
# it automatically rebuilds
build_version := $(shell shasum -a 256 $(build_path)/build-image/Dockerfile | head -c 10)

# Minikube executable
MINIKUBE ?= minikube
# Default minikube driver
MINIKUBE_DRIVER ?= virtualbox
# set docker env for minikube
MINIKUBE_DOCKER_ENV ?= eval $$($(MINIKUBE) docker-env)

# minikube shell mount for certificates
minikube_cert_mount := ~/.minikube:$(HOME)/.minikube

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# Does nothing
minikube-post-start:
