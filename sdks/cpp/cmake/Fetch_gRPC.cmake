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

cmake_minimum_required (VERSION 3.12.0)

include(FetchContent)

set(GRPC_GIT_REPO "https://github.com/grpc/grpc.git")
set(GRPC_GIT_TAG "v${AGONES_GRPC_VERSION}")

FetchContent_Declare(
    grpc
    GIT_REPOSITORY      "${GRPC_GIT_REPO}"
    GIT_TAG             "${GRPC_GIT_TAG}"
    PREFIX              grpc
    SOURCE_DIR          "grpc/.src"
    BINARY_DIR          "grpc/.bin"
    INSTALL_DIR         "grpc/.install"
    SUBBUILD_DIR        "grpc/.subbuild"
    CONFIGURE_COMMAND   ""
    BUILD_COMMAND       ""
    INSTALL_COMMAND     ""
    TEST_COMMAND        ""
)
FetchContent_GetProperties(grpc)
if (NOT grpc_POPULATED)
  message("Fetching gRPC ${AGONES_GRPC_VERSION}")
  FetchContent_Populate(
    grpc
    QUIET
    GIT_REPOSITORY      "${GRPC_GIT_REPO}"
    GIT_TAG             "${GRPC_GIT_TAG}"
    PREFIX              grpc
    SOURCE_DIR          "grpc/.src"
    BINARY_DIR          "grpc/.bin"
    INSTALL_DIR         "grpc/.install"
    SUBBUILD_DIR        "grpc/.subbuild"
    CONFIGURE_COMMAND   ""
    BUILD_COMMAND       ""
    INSTALL_COMMAND     ""
    TEST_COMMAND        ""
  )
endif()
