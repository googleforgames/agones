# Copyright 2019 Google Inc. All Rights Reserved.
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
FROM golang:latest

RUN apt-get update && \
    apt-get install -y wget jq && \
    apt-get clean

# install go
WORKDIR /usr/local
ENV GO_VERSION=1.12
ENV GO111MODULE=on
ENV GOPATH /go
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz && mkdir -p ${GOPATH}

WORKDIR /go/src/agones.dev/agones
ENV PATH /usr/local/go/bin:/go/bin:$PATH

RUN curl -sL https://deb.nodesource.com/setup_11.x | bash - && \
    apt-get install -y nodejs

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh