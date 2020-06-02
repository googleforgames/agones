#!/usr/bin/env bash

# Copyright 2018 Google LLC All Rights Reserved.
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

FEATURES=$1
echo $FEATURES
set -e
echo "installing current release"
DOCKER_RUN= make install FEATURE_GATES='"'$FEATURES'"'
echo "starting e2e test"
DOCKER_RUN= make test-e2e ARGS=-parallel=64 FEATURE_GATES='"'$FEATURES'"'
echo "completed e2e test"