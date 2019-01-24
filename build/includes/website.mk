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


#  __        __   _         _ _
#  \ \      / /__| |__  ___(_) |_ ___
#   \ \ /\ / / _ \ '_ \/ __| | __/ _ \
#    \ V  V /  __/ |_) \__ \ | |_  __/
#     \_/\_/ \___|_.__/|___/_|\__\___|
#

#
# Website targets
#

# generate the latest website
site-server: ARGS ?=-F
site-server: ENV ?= RELEASE_VERSION="$(base_version)"
site-server: ensure-build-image
	docker run --rm $(common_mounts) --workdir=$(mount_path)/site $(DOCKER_RUN_ARGS) -p 1313:1313 $(build_tag) bash -c \
	"$(ENV) hugo server --watch --baseURL=http://localhost:1313/ --bind=0.0.0.0 $(ARGS)"

site-static: ensure-build-image
	-docker run --rm $(common_mounts) --workdir=$(mount_path)/site $(DOCKER_RUN_ARGS) $(build_tag) rm -r ./public
	-mkdir $(agones_path)/site/public
	# for some reason, this only work locally
	docker run --rm $(common_mounts) --workdir=$(mount_path)/site $(DOCKER_RUN_ARGS) $(build_tag) \
		bash -c "npm list postcss-cli || npm install postcss-cli"
	docker run --rm $(common_mounts) --workdir=$(mount_path)/site $(DOCKER_RUN_ARGS) $(build_tag) \
		bash -c "npm list autoprefixer || npm install autoprefixer"
	docker run --rm $(common_mounts) --workdir=$(mount_path)/site $(DOCKER_RUN_ARGS) $(build_tag) bash -c \
		"$(ENV) hugo --config=config.toml $(ARGS)"

site-gen-app-yaml: SERVICE ?= default
site-gen-app-yaml:
	docker run --rm $(common_mounts) --workdir=$(mount_path)/site $(DOCKER_RUN_ARGS) $(build_tag) bash -c \
			"SERVICE=$(SERVICE) envsubst < app.yaml > .app.yaml"

site-deploy: site-gen-app-yaml site-static
	docker run --rm $(common_mounts) --workdir=$(mount_path)/site $(DOCKER_RUN_ARGS) $(build_tag) \
		gcloud app deploy .app.yaml --no-promote --version=$(shell git rev-parse --short=7 HEAD)

site-static-preview:
	$(MAKE) site-static ARGS="-F" ENV=RELEASE_VERSION=$(base_version)

site-deploy-preview: site-static-preview
	$(MAKE) site-deploy SERVICE=preview

site-test:
	docker run --rm --name=agones-website $(common_mounts) --workdir=$(mount_path)/site $(DOCKER_RUN_ARGS) $(build_tag) \
    	hugo server --watch --baseURL="http://localhost:1313/site/" &
	until docker exec agones-website curl -o /dev/null --silent http://localhost:1313/site/; \
			do \
				echo "Waiting for server to start..."; \
				sleep 1; \
			done
	( trap 'docker stop agones-website' EXIT; docker exec agones-website linkchecker --anchors http://localhost:1313/site/ )