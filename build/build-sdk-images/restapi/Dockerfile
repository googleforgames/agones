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
ARG BASE_IMAGE=agones-build-sdk-base:latest
FROM $BASE_IMAGE

RUN apt-get --allow-releaseinfo-change update && \
    apt-get install -y wget jq software-properties-common gnupg && \
    apt-get clean

# install go
WORKDIR /usr/local
ENV GO_VERSION=1.19.1
ENV GO111MODULE=on
ENV GOPATH /go
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz && mkdir -p ${GOPATH}

RUN echo '§' && apt-get -qy update 
RUN add-apt-repository -y -r ppa:chris-lea/node.js
RUN rm -f /etc/apt/sources.list.d/chris-lea-node_js-*.list
RUN rm -f /etc/apt/sources.list.d/chris-lea-node_js-*.list.save

ARG KEYRING=/usr/share/keyrings/nodesource.gpg
ARG VERSION=node_16.x

RUN curl -fsSL https://deb.nodesource.com/gpgkey/nodesource.gpg.key | gpg --dearmor | tee "$KEYRING" >/dev/null
RUN gpg --no-default-keyring --keyring "$KEYRING" --list-keys

ARG DISTRO="buster"
RUN echo "deb [signed-by=$KEYRING] https://deb.nodesource.com/$VERSION $DISTRO main" | tee /etc/apt/sources.list.d/nodesource.list
RUN echo "deb-src [signed-by=$KEYRING] https://deb.nodesource.com/$VERSION $DISTRO main" | tee -a /etc/apt/sources.list.d/nodesource.list

RUN apt-get update && apt-get install -y nodejs

RUN apt-get install -qq -y openjdk-11-jre > /dev/null

ENV PATH /usr/local/go/bin:/go/bin:$PATH

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh