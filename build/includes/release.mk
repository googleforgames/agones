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
	$(MAKE) ENV=HUGO_ENV=snapshot site-deploy SERVICE=$$version

# creates release-branch
release-branch: RELEASE_VERSION ?= $(base_version)
release-branch: $(ensure-build-image)
	@echo "Starting release for version: $(RELEASE_VERSION)"

	# switch to the right project
	$(DOCKER_RUN) gcloud config configurations activate agones-images
    
	git remote update -p
	git fetch --all --tags
	git checkout -b release-$(RELEASE_VERSION) v$(RELEASE_VERSION)
	git push -u upstream release-$(RELEASE_VERSION)

	@echo "Now go make the $(RELEASE_VERSION) release on Github!"

# push the current chart to google cloud storage and update the index
push-chart: $(ensure-build-image) build-chart
	docker run $(DOCKER_RUN_ARGS) --rm $(common_mounts) -w $(workdir_path) $(build_tag) bash -c \
		"gsutil copy gs://$(GCP_BUCKET_CHARTS)/index.yaml ./install/helm/bin/index.yaml || /bin/true && \
		helm repo index --merge ./install/helm/bin/index.yaml ./install/helm/bin && \
		cat ./install/helm/bin/index.yaml && ls ./install/helm/bin/ && \
		cp ./install/helm/bin/index.yaml ./install/helm/bin/index-$(VERSION).yaml && \
		gsutil copy ./install/helm/bin/*.* gs://$(GCP_BUCKET_CHARTS)/"

# Ensure the example images exists
pre-build-release:
	docker run --rm $(common_mounts) -w $(workdir_path)/build/release $(build_tag) \
		gcloud builds submit . --config=./pre_cloudbuild.yaml $(ARGS)

# Build and push the images in the release repository,
# stores artifacts,
# Pushes the current chart version to the helm repository hosted on gcs.
post-build-release:
	docker run --rm $(common_mounts) -w $(workdir_path)/build/release $(build_tag) \
		gcloud builds submit . --substitutions _VERSION=$(base_version) --config=./post_cloudbuild.yaml $(ARGS)
