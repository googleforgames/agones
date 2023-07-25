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

REPOSITORY ?=

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
project_path := $(dir $(mkfile_path))
root_path := $(realpath $(project_path)/../..)
image_tag = xonotic-example:1.1

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# Build a docker image for the server, and tag it
build:
	docker build -f $(project_path)/Dockerfile --tag=$(REPOSITORY)/$(image_tag) .

# Push a docker image to the repository, and save it
push: build
	docker push $(REPOSITORY)/$(image_tag)

# check if hosted on Google Artifact Registry
gar-check:
	gcloud container images describe $(REPOSITORY)/$(image_tag)

#output the tag for this image
echo-image-tag:
	@echo $(REPOSITORY)/$(image_tag)

# build and push the xonotic image with specified tag
cloud-build:
	gcloud builds submit --config=cloudbuild.yaml --substitutions=_IMAGE_TAG=$(image_tag)
