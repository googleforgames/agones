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

#
# Include for managing the examples
#

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# test all example images exist on Google Cloud Registry
test-examples-on-gcr: example-image-test.autoscaler-webhook
test-examples-on-gcr: example-image-test.cpp-simple
test-examples-on-gcr: example-image-test.nodejs-simple
test-examples-on-gcr: example-image-test.rust-simple
test-examples-on-gcr: example-image-test.unity-simple
test-examples-on-gcr: example-image-test.xonotic
test-examples-on-gcr: example-image-test.crd-client
test-examples-on-gcr: example-image-test.supertuxkart
test-examples-on-gcr: example-image-test.simple-game-server

# Test to ensure the example image found in the % folder is on GCR. Fails if it is not.
example-image-test.%:
	$(DOCKER_RUN) bash -c "cd examples/$* && make gcr-check"
