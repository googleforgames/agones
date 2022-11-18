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

# ForceUpdate 12 -- change here if you need to force a rebuild

FROM debian:bullseye

RUN apt-get update && \
    apt-get install -y build-essential gnupg curl git wget psmisc rsync make python bash-completion \
    zip nano jq graphviz gettext-base plantuml software-properties-common && \
    apt-get clean

# install go
WORKDIR /usr/local
ENV GO_VERSION=1.19.1
ENV GOPATH /go
ENV GO111MODULE=on
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz && mkdir ${GOPATH}

# install gcloud + kubectl, because it's an easy way to test/dev against kubernetes.
WORKDIR /opt

# credits https://cloud.google.com/sdk/docs/install#deb
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key --keyring /usr/share/keyrings/cloud.google.gpg  add - && apt-get update -y && \
    apt-get install google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin google-cloud-cli-app-engine-go -y && \
    echo "source /usr/share/google-cloud-sdk/completion.bash.inc" >> /root/.bashrc

# update the path for go
ENV PATH /usr/local/go/bin:/go/bin:$PATH

# install go tooling for development, building and testing
RUN go install golang.org/x/tools/cmd/goimports@latest

# the kubernetes version for the file
ENV KUBERNETES_VER 1.23.9

# overwrite kubectl as we want a specific version
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VER}/bin/linux/amd64/kubectl && \
    chmod go+rx ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl
RUN echo "source <(kubectl completion bash)" >> /root/.bashrc

# install the release branch of the code generator tools
RUN mkdir -p /go/src/k8s.io && cd /go/src/k8s.io && \
    git clone -b kubernetes-${KUBERNETES_VER} --depth=3 https://github.com/kubernetes/code-generator.git

# install Helm package manager
ENV HELM_VER 3.5.0
ENV HELM_URL https://get.helm.sh/helm-v${HELM_VER}-linux-amd64.tar.gz
RUN curl -L  ${HELM_URL} > /tmp/helm.tar.gz \
    && tar -zxvf /tmp/helm.tar.gz -C /tmp \
    && mv /tmp/linux-amd64/helm /usr/local/bin/helm \
    && chmod go+rx /usr/local/bin/helm \
    && rm /tmp/helm.tar.gz && rm -rf /tmp/linux-amd64
RUN echo "source <(helm completion bash)" >> /root/.bashrc

# install golang-ci linter
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0

#
#  \ \      / /__| |__  ___(_) |_ ___
#   \ \ /\ / / _ \ '_ \/ __| | __/ _ \
#    \ V  V /  __/ |_) \__ \ | |_  __/
#     \_/\_/ \___|_.__/|___/_|\__\___|
#

ENV HUGO_VER 0.82.1
RUN mkdir /tmp/hugo && \
    wget -q -O /tmp/hugo/hugo.tar.gz https://github.com/gohugoio/hugo/releases/download/v${HUGO_VER}/hugo_extended_${HUGO_VER}_Linux-64bit.tar.gz && \
    tar -zxvf /tmp/hugo/hugo.tar.gz -C /tmp/hugo/ && \
    mv /tmp/hugo/hugo /usr/local/bin/ && \
    rm -r /tmp/hugo

RUN add-apt-repository -y -r ppa:chris-lea/node.js
RUN rm -f /etc/apt/sources.list.d/chris-lea-node_js-*.list
RUN rm -f /etc/apt/sources.list.d/chris-lea-node_js-*.list.save

ARG KEYRING=/usr/share/keyrings/nodesource.gpg
ARG VERSION=node_18.x

RUN curl -fsSL https://deb.nodesource.com/gpgkey/nodesource.gpg.key | gpg --dearmor | tee "$KEYRING" >/dev/null
RUN gpg --no-default-keyring --keyring "$KEYRING" --list-keys

ARG DISTRO="buster"
RUN echo "deb [signed-by=$KEYRING] https://deb.nodesource.com/$VERSION $DISTRO main" | tee /etc/apt/sources.list.d/nodesource.list
RUN echo "deb-src [signed-by=$KEYRING] https://deb.nodesource.com/$VERSION $DISTRO main" | tee -a /etc/apt/sources.list.d/nodesource.list

RUN apt-get update && apt-get install -y nodejs

# install API reference docs generator
RUN mkdir -p /go/src/github.com/ahmetb && \
    cd /go/src/github.com/ahmetb && git clone -b v0.2.0 https://github.com/ahmetb/gen-crd-api-reference-docs && \
    cd ./gen-crd-api-reference-docs && go build

# html checker
RUN mkdir /tmp/htmltest && \
    wget -O /tmp/htmltest/htmltest.tar.gz https://github.com/wjdp/htmltest/releases/download/v0.13.0/htmltest_0.13.0_linux_amd64.tar.gz && \
    tar -zxvf /tmp/htmltest/htmltest.tar.gz -C /tmp/htmltest && \
    mv /tmp/htmltest/htmltest /usr/local/bin && \
    rm -r /tmp/htmltest

# make sure we keep the path to go
RUN echo "export PATH=/usr/local/go/bin:/go/bin/:\$PATH" >> /root/.bashrc
# make nano the editor
RUN echo "export EDITOR=nano" >> /root/.bashrc

# install terraform
RUN wget -nv -O terraform.zip https://releases.hashicorp.com/terraform/1.0.8/terraform_1.0.8_linux_amd64.zip && unzip ./terraform.zip && mv terraform /usr/local/bin/

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh

WORKDIR /go
