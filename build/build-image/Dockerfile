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
# Base Stage - Build Tools
# Use a minimal base image to reduce attack surface
FROM debian:bookworm-slim AS base

# Install build dependencies in one layer
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential gnupg curl git wget psmisc rsync make python3 bash-completion \
    zip nano jq graphviz gettext-base plantuml software-properties-common ruby-full \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Install fpm, create dummy package, and clean up
RUN gem install fpm && \
    fpm -s empty -t deb -p python2.7-0.0.0-0-any.deb --name python2.7 --license MIT --version 0.0.0 --architecture all --description 'Dummy package to satisfy google-cloud-cli dependencies.' --maintainer 'Yourself' && \
    dpkg -i python2.7-0.0.0-0-any.deb && \
    rm python2.7-0.0.0-0-any.deb

# Install Go
FROM base AS go-build
WORKDIR /usr/local
ENV GO_VERSION=1.21.12
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz

# Install gcloud, kubectl, Helm, and other tools
FROM base AS tools
WORKDIR /opt

# Install Google Cloud CLI and kubectl
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin google-cloud-cli-app-engine-go && \
    rm -rf /var/lib/apt/lists/* && \
    echo "source /usr/share/google-cloud-sdk/completion.bash.inc" >> /root/.bashrc

# Install kubectl
ENV KUBERNETES_VER 1.29.7
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VER}/bin/linux/amd64/kubectl && \
    chmod go+rx ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl && \
    echo "source <(kubectl completion bash)" >> /root/.bashrc

# Install Helm
ENV HELM_VER 3.10.3
ENV HELM_URL https://get.helm.sh/helm-v${HELM_VER}-linux-amd64.tar.gz
RUN curl -L ${HELM_URL} | tar -xzC /tmp && \
    mv /tmp/linux-amd64/helm /usr/local/bin/helm && \
    chmod go+rx /usr/local/bin/helm && \
    rm -rf /tmp/linux-amd64 && \
    echo "source <(helm completion bash)" >> /root/.bashrc

# Install GolangCI linter, yamllint, gotestsum, Hugo, Terraform, and other tools
FROM go-build AS final
WORKDIR /root

# Install Go tools
RUN go install golang.org/x/tools/cmd/goimports@latest && \
    go install golang.org/x/pkgsite/cmd/pkgsite@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.2 && \
    go install gotest.tools/gotestsum@latest

# Install yamllint
RUN apt-get update && \
    apt-get install -y --no-install-recommends yamllint && \
    rm -rf /var/lib/apt/lists/*

# Install Hugo
ENV HUGO_VER 0.118.2
RUN wget -q -O hugo.tar.gz https://github.com/gohugoio/hugo/releases/download/v${HUGO_VER}/hugo_extended_${HUGO_VER}_Linux-64bit.tar.gz && \
    tar -zxvf hugo.tar.gz -C /usr/local/bin/ && \
    rm hugo.tar.gz

# Install Node.js
ARG NODE_MAJOR=20
RUN curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" > /etc/apt/sources.list.d/nodesource.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends nodejs && \
    rm -rf /var/lib/apt/lists/*

# Install additional tools
RUN wget -nv -O terraform.zip https://releases.hashicorp.com/terraform/1.5.4/terraform_1.5.4_linux_amd64.zip && \
    unzip terraform.zip && \
    mv terraform /usr/local/bin/ && \
    rm terraform.zip

# Install code generator tools
RUN mkdir -p /go/src/github.com/ahmetb && \
    cd /go/src/github.com/ahmetb && git clone -b v0.2.0 https://github.com/ahmetb/gen-crd-api-reference-docs && \
    cd ./gen-crd-api-reference-docs && go build

# Install htmltest
RUN wget -O htmltest.tar.gz https://github.com/wjdp/htmltest/releases/download/v0.13.0/htmltest_0.13.0_linux_amd64.tar.gz && \
    tar -zxvf htmltest.tar.gz -C /tmp && \
    mv /tmp/htmltest /usr/local/bin && \
    rm -rf /tmp/htmltest htmltest.tar.gz

# Copy scripts and set environment
COPY *.sh /root/
RUN chmod +x /root/*.sh
ENV PATH /usr/local/go/bin:/go/bin:$PATH
RUN echo "export EDITOR=nano" >> /root/.bashrc && \
    echo "export PATH=/usr/local/go/bin:/go/bin:$PATH" >> /root/.bashrc

# Final working directory
WORKDIR /go
