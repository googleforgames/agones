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

# Stage 1: Build Tools
FROM debian:bookworm AS build-env

# Set up arguments for tool versions
ARG GO_VERSION=1.22.6
ARG KUBERNETES_VER=1.29.7
ARG HELM_VER=3.10.3
ARG HUGO_VER=0.118.2
ARG NODE_MAJOR=20

# Install base build tools and dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential gnupg curl git wget psmisc rsync make python3 bash-completion \
    zip nano jq graphviz gettext-base plantuml software-properties-common ruby-full \
    ca-certificates unzip gnupg2 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Create a dummy python2.7 package using fpm for google-cloud-cli dependencies
RUN gem install fpm && \
    fpm -s empty -t deb -p /opt/python2.7-0.0.0-0-any.deb --name python2.7 --license MIT --version 0.0.0 \
    --architecture all --description 'Dummy package to satisfy google-cloud-cli dependencies.' --maintainer 'Yourself' && \
    dpkg -i /opt/python2.7-0.0.0-0-any.deb && rm /opt/python2.7-0.0.0-0-any.deb

# Install Go
WORKDIR /usr/local
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz

ENV GOPATH /go
ENV GO111MODULE=on
ENV PATH /usr/local/go/bin:/go/bin:$PATH

# Install Google Cloud SDK, kubectl, and additional Go tools
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | \
    tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin google-cloud-cli-app-engine-go && \
    curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VER}/bin/linux/amd64/kubectl && \
    chmod +x kubectl && mv kubectl /usr/local/bin/kubectl && \
    go install golang.org/x/tools/cmd/goimports@latest && \
    go install golang.org/x/pkgsite/cmd/pkgsite@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0 && \
    go install gotest.tools/gotestsum@latest

# Install Helm, Hugo, and Terraform
RUN curl -L https://get.helm.sh/helm-v${HELM_VER}-linux-amd64.tar.gz -o /tmp/helm.tar.gz && \
    tar -zxvf /tmp/helm.tar.gz -C /tmp && \
    mv /tmp/linux-amd64/helm /usr/local/bin/helm && rm -rf /tmp/helm* && \
    wget -q https://github.com/gohugoio/hugo/releases/download/v${HUGO_VER}/hugo_extended_${HUGO_VER}_Linux-64bit.tar.gz -O /tmp/hugo.tar.gz && \
    tar -zxvf /tmp/hugo.tar.gz -C /tmp && mv /tmp/hugo /usr/local/bin/ && rm -rf /tmp/hugo* && \
    wget -nv -O terraform.zip https://releases.hashicorp.com/terraform/1.5.4/terraform_1.5.4_linux_amd64.zip && \
    unzip terraform.zip && mv terraform /usr/local/bin/terraform && rm terraform.zip

# Install Node.js and npm
RUN mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" > /etc/apt/sources.list.d/nodesource.list && \
    apt-get update && apt-get install -y --no-install-recommends nodejs

# Install API reference docs generator
RUN mkdir -p /go/src/github.com/ahmetb && \
    git clone https://github.com/ahmetb/gen-crd-api-reference-docs.git /go/src/github.com/ahmetb/gen-crd-api-reference-docs && \
    cd /go/src/github.com/ahmetb/gen-crd-api-reference-docs && go build

# Install HTML checker
RUN mkdir /tmp/htmltest && \
    wget -O /tmp/htmltest/htmltest.tar.gz https://github.com/wjdp/htmltest/releases/download/v0.13.0/htmltest_0.13.0_linux_amd64.tar.gz && \
    tar -zxvf /tmp/htmltest/htmltest.tar.gz -C /tmp/htmltest && mv /tmp/htmltest/htmltest /usr/local/bin && rm -r /tmp/htmltest

# Copy scripts and make them executable
COPY *.sh /root/
RUN chmod +x /root/*.sh

# Stage 2: Final image with runtime dependencies
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    psmisc rsync bash-completion ca-certificates curl gnupg git yamllint && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy tools and scripts from build environment
COPY --from=build-env /usr/local/go /usr/local/go
COPY --from=build-env /usr/local/bin /usr/local/bin
COPY --from=build-env /go /go
COPY --from=build-env /root/*.sh /root/

# Set environment variables
ENV GOPATH /go
ENV GO111MODULE=on
ENV PATH /usr/local/go/bin:/go/bin:$PATH

# Set the default working directory
WORKDIR /go

# Set the default command
CMD ["/bin/bash"]
