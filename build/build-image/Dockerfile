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
FROM debian:bookworm AS build-tools

# Set up arguments for tool versions
ARG GO_VERSION=1.22.6
ARG KUBERNETES_VER=1.29.7
ARG HELM_VER=3.10.3
ARG HUGO_VER=0.118.2
ARG NODE_MAJOR=20
ARG TF_VERSION=1.5.4

# Install base build tools and dependencies in one layer
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential gnupg curl git wget psmisc rsync make python3 bash-completion \
    zip nano jq graphviz gettext-base plantuml software-properties-common ruby-full \
    ca-certificates unzip && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Create a dummy python2.7 package using fpm (for google-cloud-cli dependencies)
RUN gem install fpm && \
    fpm -s empty -t deb -p /opt/python2.7-0.0.0-0-any.deb --name python2.7 --license MIT --version 0.0.0 \
    --architecture all --description 'Dummy package to satisfy google-cloud-cli dependencies.' --maintainer 'Youssef' && \
    dpkg -i /opt/python2.7-0.0.0-0-any.deb && rm /opt/python2.7-0.0.0-0-any.deb

# Install Go
WORKDIR /usr/local
ENV GOPATH /go
ENV GO111MODULE=on
ENV PATH /usr/local/go/bin:/go/bin:$PATH
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz -C /usr/local && \
    rm go${GO_VERSION}.linux-amd64.tar.gz && mkdir -p ${GOPATH}/bin

# Install Google Cloud SDK and dependencies in a single layer
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | \
    tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin google-cloud-cli-app-engine-go && \
    echo "source /usr/share/google-cloud-sdk/completion.bash.inc" >> /root/.bashrc

# Install additional Go tools
RUN go install golang.org/x/tools/cmd/goimports@latest || echo "Failed to install goimports" && \
    go install golang.org/x/pkgsite/cmd/pkgsite@latest || echo "Failed to install pkgsite" && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0 || echo "Failed to install golangci-lint" && \
    go install gotest.tools/gotestsum@latest || echo "Failed to install gotestsum"

# Install specific kubectl version
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VER}/bin/linux/amd64/kubectl && \
    chmod +x kubectl && mv kubectl /usr/local/bin/kubectl && \
    echo "source <(kubectl completion bash)" >> /root/.bashrc

# Install the release branch of the code generator tools
RUN mkdir -p /go/src/k8s.io && cd /go/src/k8s.io && \
    git clone -b v0.30.0 --depth=3 https://github.com/kubernetes/code-generator.git

# Install Helm
RUN curl -L https://get.helm.sh/helm-v${HELM_VER}-linux-amd64.tar.gz -o /tmp/helm.tar.gz && \
    tar -zxvf /tmp/helm.tar.gz -C /tmp && \
    mv /tmp/linux-amd64/helm /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm && rm -rf /tmp/helm*

# Install Hugo
RUN wget -q -O /tmp/hugo.tar.gz https://github.com/gohugoio/hugo/releases/download/v${HUGO_VER}/hugo_extended_${HUGO_VER}_Linux-64bit.tar.gz && \
    tar -zxvf /tmp/hugo.tar.gz -C /tmp && \
    mv /tmp/hugo /usr/local/bin/hugo && rm -rf /tmp/hugo*

# Install Node.js and npm
RUN mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" > /etc/apt/sources.list.d/nodesource.list && \
    apt-get update && apt-get install -y --no-install-recommends nodejs

# Install Terraform
RUN wget -nv -O /tmp/terraform.zip https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_linux_amd64.zip && \
    unzip /tmp/terraform.zip -d /usr/local/bin/ && rm /tmp/terraform.zip

# Copy scripts and make them executable
COPY *.sh /root/
RUN chmod +x /root/*.sh

# Stage 2: Runtime Environment
FROM debian:bookworm-slim AS runtime

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    psmisc rsync bash-completion ca-certificates curl gnupg git yamllint && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy binaries from build-tools stage
COPY --from=build-tools /usr/local/go /usr/local/go
COPY --from=build-tools /usr/local/bin/helm /usr/local/bin/helm
COPY --from=build-tools /usr/local/bin/hugo /usr/local/bin/hugo
COPY --from=build-tools /go/bin /usr/local/bin

# Set environment variables
ENV GOPATH /go
ENV GO111MODULE=on
ENV PATH /usr/local/go/bin:/go/bin:$PATH

# Set the default working directory
WORKDIR /go

# Set the default command
CMD ["/bin/bash"]