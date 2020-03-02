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

# ForceUpdate 11 -- change here if you need to force a rebuild

FROM debian:stretch

RUN apt-get update && \
    apt-get install -y build-essential gnupg curl git wget psmisc rsync make python bash-completion \
    zip nano jq graphviz gettext-base plantuml && \
    apt-get clean

# install go
WORKDIR /usr/local
ENV GO_VERSION=1.13.5
ENV GOPATH /go
ENV GO111MODULE=on
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz && mkdir ${GOPATH}

# install gcloud + kubectl, because it's an easy way to test/dev against kubernetes.
WORKDIR /opt
RUN wget -q https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.zip && unzip -q google-cloud-sdk.zip && \
    rm google-cloud-sdk.zip && \
    /opt/google-cloud-sdk/install.sh --usage-reporting=true --path-update=true --bash-completion=true --rc-path=/root/.bashrc

# update the path for both go and gcloud
ENV PATH /usr/local/go/bin:/go/bin:/opt/google-cloud-sdk/bin:$PATH

# install go tooling for development, building and testing
RUN go get -u golang.org/x/tools/cmd/goimports

# RUN gcloud components update
RUN gcloud components update && gcloud components install app-engine-go

# the kubernetes version for the file
ENV KUBERNETES_VER 1.14.10

# overwrite kubectl as we want a specific version
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VER}/bin/linux/amd64/kubectl && \
    chmod go+rx ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl
RUN echo "source <(kubectl completion bash)" >> /root/.bashrc

# install the release branch of the code generator tools
RUN mkdir -p /go/src/k8s.io && cd /go/src/k8s.io && \
    git clone -b kubernetes-${KUBERNETES_VER} --depth=3 https://github.com/kubernetes/code-generator.git

# install Helm package manager
ENV HELM_VER 2.11.0
ENV HELM_URL https://storage.googleapis.com/kubernetes-helm/helm-v${HELM_VER}-linux-amd64.tar.gz
RUN curl -L  ${HELM_URL} > /tmp/helm.tar.gz \
    && tar -zxvf /tmp/helm.tar.gz -C /tmp \
    && mv /tmp/linux-amd64/helm /usr/local/bin/helm \
    && chmod go+rx /usr/local/bin/helm \
    && rm /tmp/helm.tar.gz && rm -rf /tmp/linux-amd64
RUN echo "source <(helm completion bash)" >> /root/.bashrc

# install golang-ci linter
RUN curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $GOPATH/bin v1.22.2

#
#  \ \      / /__| |__  ___(_) |_ ___
#   \ \ /\ / / _ \ '_ \/ __| | __/ _ \
#    \ V  V /  __/ |_) \__ \ | |_  __/
#     \_/\_/ \___|_.__/|___/_|\__\___|
#

ENV HUGO_VER 0.58.3
RUN mkdir /tmp/hugo && \
    wget -q -O /tmp/hugo/hugo.tar.gz https://github.com/gohugoio/hugo/releases/download/v${HUGO_VER}/hugo_extended_${HUGO_VER}_Linux-64bit.tar.gz && \
    tar -zxvf /tmp/hugo/hugo.tar.gz -C /tmp/hugo/ && \
    mv /tmp/hugo/hugo /usr/local/bin/ && \
    rm -r /tmp/hugo

RUN curl -sL https://deb.nodesource.com/setup_11.x | bash - && \
    apt-get install -y nodejs

# install API reference docs generator
RUN mkdir -p /go/src/github.com/ahmetb && \
    cd /go/src/github.com/ahmetb && git clone -b v0.1.1 https://github.com/ahmetb/gen-crd-api-reference-docs && \
    cd ./gen-crd-api-reference-docs && go build

# html checker
RUN mkdir /tmp/htmltest && \
    wget -O /tmp/htmltest/htmltest.tar.gz https://github.com/wjdp/htmltest/releases/download/v0.10.1/htmltest_0.10.1_linux_amd64.tar.gz && \
    tar -zxvf /tmp/htmltest/htmltest.tar.gz -C /tmp/htmltest && \
    mv /tmp/htmltest/htmltest /usr/local/bin && \
    rm -r /tmp/htmltest

# make sure we keep the path to go
RUN echo "export PATH=/usr/local/go/bin:/go/bin/:\$PATH" >> /root/.bashrc
# make nano the editor
RUN echo "export EDITOR=nano" >> /root/.bashrc

# install terraform
RUN wget -nv -O terraform.zip https://releases.hashicorp.com/terraform/0.12.21/terraform_0.12.21_linux_amd64.zip && unzip ./terraform.zip && mv terraform /usr/local/bin/

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh

WORKDIR /go
