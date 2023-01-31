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
ARG BASE_IMAGE=agones-build-sdk-base:latest
FROM $BASE_IMAGE

RUN apt-get --allow-releaseinfo-change update && \
    apt-get install -y zip wget clang-format && \
    apt-get clean

RUN wget -q https://cmake.org/files/v3.14/cmake-3.14.1-Linux-x86_64.sh -O /cmake-3.14.1-Linux-x86_64.sh
RUN mkdir /opt/cmake
RUN sh /cmake-3.14.1-Linux-x86_64.sh --prefix=/opt/cmake --skip-license
RUN ln -s /opt/cmake/bin/cmake /usr/local/bin/cmake
RUN cmake --version

WORKDIR /go/src/agones.dev/agones

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh
