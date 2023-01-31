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

#   ____      _
#  |  _ \ ___| | ___  __ _ ___  ___
#  | |_) / _ \ |/ _ \/ _` / __|/ _ \
#  |  _ <  __/ |  __/ (_| \__ \  __/
#  |_| \_\___|_|\___|\__,_|___/\___|
#

#
# targets for an Agones release
#

# agones image release registry
release_registry = us-docker.pkg.dev/agones-images/release

# generate a changelog using github-changelog-generator
gen-changelog: RELEASE_VERSION ?= $(base_version)
gen-changelog: RELEASE_BRANCH ?= main
gen-changelog:
	read -p 'Github Token: ' TOKEN && \
    docker run -it --rm -v "$(agones_path)":/usr/local/src/your-app ferrarimarco/github-changelog-generator:1.15.0 \
		--user=googleforgames --project=agones \
		--bug-labels=kind/bug --enhancement-labels=kind/feature \
		--breaking-labels=kind/breaking --security-labels=area/security \
		--future-release "v$(RELEASE_VERSION)" \
		--release-branch=$(RELEASE_BRANCH) \
		--token $$TOKEN

# outputs the markdown for the example images section of the release template
release-example-image-markdown: example-image-markdown.allocation-endpoint
release-example-image-markdown: example-image-markdown.autoscaler-webhook
release-example-image-markdown: example-image-markdown.cpp-simple
release-example-image-markdown: example-image-markdown.crd-client
release-example-image-markdown: example-image-markdown.nodejs-simple
release-example-image-markdown: example-image-markdown.rust-simple
release-example-image-markdown: example-image-markdown.simple-game-server
release-example-image-markdown: example-image-markdown.supertuxkart
release-example-image-markdown: example-image-markdown.unity-simple
release-example-image-markdown: example-image-markdown.xonotic

example-image-markdown.%:
	@cd $(agones_path)/examples/$* && \
    tag=$$(make -silent echo-image-tag) && \
    echo "- [$$tag](https://$$tag)"


# Deploys the site by taking in the base version and deploying the previous version
release-deploy-site: $(ensure-build-image)
release-deploy-site: DOCKER_RUN_ARGS += -e GOFLAGS="-mod=mod" --entrypoint=/usr/local/go/bin/go
release-deploy-site:
	version=$$($(DOCKER_RUN) run $(mount_path)/build/scripts/previousversion/main.go -version $(base_version)) && \
	echo "Deploying Site Version: $$version" && \
	$(MAKE) site-deploy SERVICE=$$version

# Creates a release. Version defaults to the base_version
# - Checks out a release branch
# - Build binaries and images
# - Creates sdk and binary archives, and moves the into the /release folder for upload
# - Creates a zip of the install.yaml, LICENCE and README.md for installation
# - Pushes the current chart version to the helm repository hosted on gcs.
do-release: RELEASE_VERSION ?= $(base_version)
do-release: $(ensure-build-image)
	@echo "Starting release for version: $(RELEASE_VERSION)"

	# switch to the right project
	$(DOCKER_RUN) gcloud config configurations activate agones-images

	git checkout -b release-$(RELEASE_VERSION)
	-rm -rf $(agones_path)/release
	mkdir $(agones_path)/release

	$(MAKE) -j 4 build VERSION=$(RELEASE_VERSION) REGISTRY=$(release_registry) FULL_BUILD=1
	cp $(agones_path)/cmd/sdk-server/bin/agonessdk-server-$(RELEASE_VERSION).zip $(agones_path)/release
	cp $(agones_path)/sdks/cpp/.archives/agonessdk-$(RELEASE_VERSION)-linux-arch_64.tar.gz $(agones_path)/release
	cd $(agones_path) &&  zip -r ./release/agones-install-$(RELEASE_VERSION).zip ./README.md ./install ./LICENSE

	$(MAKE) gcloud-auth-docker
	$(MAKE) -j 4 push REGISTRY=$(release_registry) VERSION=$(RELEASE_VERSION)

	$(MAKE) push-chart VERSION=$(RELEASE_VERSION)
	git push -u upstream release-$(RELEASE_VERSION)

	@echo "Now go make the $(RELEASE_VERSION) release on Github!"

build-release: 
	cd $(agones_path) && gcloud builds submit . --substitutions _VERSION=$(base_version) --config=./build/release/cloudbuild.yaml $(ARGS)

