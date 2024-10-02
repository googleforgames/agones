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
# Build Stage - Build Tools and Dependencies
FROM debian:bookworm AS build-tools

# Install base build tools and dependencies
RUN apt-get update && \
    apt-get install -y build-essential gnupg curl git wget psmisc rsync make python3 bash-completion \
    zip nano jq graphviz gettext-base plantuml software-properties-common ruby-full && \
    apt-get clean

# Create a dummy python2.7 package using fpm
RUN gem install fpm && \
    fpm -s empty -t deb -p /opt/python2.7-0.0.0-0-any.deb --name python2.7 --license MIT --version 0.0.0 --architecture all --description 'Dummy package to satisfy google-cloud-cli dependencies.' --maintainer 'Youssef'

# Install the .deb package
RUN dpkg -i /opt/python2.7-0.0.0-0-any.deb

# List contents of /opt to verify the .deb file exists
RUN ls -l /opt


# Install Go
WORKDIR /usr/local
ENV GO_VERSION=1.21.12
ENV GOPATH /go
ENV GO111MODULE=on
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz && mkdir ${GOPATH}

# Install Google Cloud SDK and dependencies
WORKDIR /opt
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - && \
    apt-get update -y && \
    apt-get install -y google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin google-cloud-cli-app-engine-go && \
    echo "source /usr/share/google-cloud-sdk/completion.bash.inc" >> /root/.bashrc

# Install Go tools and additional binaries
ENV PATH /usr/local/go/bin:/go/bin:$PATH
RUN go install golang.org/x/tools/cmd/goimports@latest && \
    go install golang.org/x/pkgsite/cmd/pkgsite@latest

# Install specific kubectl version
ENV KUBERNETES_VER 1.29.7
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VER}/bin/linux/amd64/kubectl && \
    chmod go+rx ./kubectl && mv ./kubectl /usr/local/bin/kubectl && \
    echo "source <(kubectl completion bash)" >> /root/.bashrc

# Clone Kubernetes code generator tools
RUN mkdir -p /go/src/k8s.io && cd /go/src/k8s.io && \
    git clone -b v0.30.0 --depth=3 https://github.com/kubernetes/code-generator.git

# Install Helm package manager
ENV HELM_VER 3.10.3
RUN curl -L https://get.helm.sh/helm-v${HELM_VER}-linux-amd64.tar.gz -o /tmp/helm.tar.gz && \
    tar -zxvf /tmp/helm.tar.gz -C /tmp && mv /tmp/linux-amd64/helm /usr/local/bin/helm && \
    chmod go+rx /usr/local/bin/helm && rm /tmp/helm.tar.gz && rm -rf /tmp/linux-amd64

# Install golangci-lint
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.2

# Install yamllint
RUN apt-get update && apt-get install -y yamllint

# Install gotestsum
RUN go install gotest.tools/gotestsum@latest

# Install Hugo
ENV HUGO_VER 0.118.2
RUN mkdir /tmp/hugo && \
    wget -q -O /tmp/hugo/hugo.tar.gz https://github.com/gohugoio/hugo/releases/download/v${HUGO_VER}/hugo_extended_${HUGO_VER}_Linux-64bit.tar.gz && \
    tar -zxvf /tmp/hugo/hugo.tar.gz -C /tmp/hugo/ && \
    mv /tmp/hugo/hugo /usr/local/bin/ && \
    rm -r /tmp/hugo

# Install Node.js
ARG NODE_MAJOR=20
RUN apt-get update && apt-get install -y ca-certificates curl gnupg && \
    mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" > /etc/apt/sources.list.d/nodesource.list && \
    apt-get update && apt-get install -y nodejs

# Install API reference docs generator
RUN mkdir -p /go/src/github.com/ahmetb && \
    cd /go/src/github.com/ahmetb && git clone -b v0.2.0 https://github.com/ahmetb/gen-crd-api-reference-docs && \
    cd ./gen-crd-api-reference-docs && go build

# Install HTML checker
RUN mkdir /tmp/htmltest && \
    wget -O /tmp/htmltest/htmltest.tar.gz https://github.com/wjdp/htmltest/releases/download/v0.13.0/htmltest_0.13.0_linux_amd64.tar.gz && \
    tar -zxvf /tmp/htmltest/htmltest.tar.gz -C /tmp/htmltest && \
    mv /tmp/htmltest/htmltest /usr/local/bin && rm -r /tmp/htmltest

# Install Terraform
RUN wget -nv -O terraform.zip https://releases.hashicorp.com/terraform/1.5.4/terraform_1.5.4_linux_amd64.zip && \
    unzip ./terraform.zip && mv terraform /usr/local/bin/ && rm terraform.zip

# Copy scripts for the build stage
COPY *.sh /root/
RUN chmod +x /root/*.sh

# Final Stage - Runtime Environment
FROM debian:bookworm-slim AS runtime

# Install runtime dependencies
RUN apt-get update && apt-get install -y psmisc rsync bash-completion ca-certificates curl gnupg git && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy dummy python2.7 package from build-tools stage
COPY --from=build-tools /opt/python2.7-0.0.0-0-any.deb /tmp/
RUN dpkg -i /tmp/python2.7-0.0.0-0-any.deb && rm /tmp/python2.7-0.0.0-0-any.deb

# Copy Go, Helm, and Hugo binaries from build-tools stage
COPY --from=build-tools /usr/local/go /usr/local/go
COPY --from=build-tools /usr/local/bin/helm /usr/local/bin/helm
COPY --from=build-tools /usr/local/bin/hugo /usr/local/bin/hugo
COPY --from=build-tools /go/bin/golangci-lint /usr/local/bin/golangci-lint
COPY --from=build-tools /go/bin /go/bin

# Set environment variables
ENV GOPATH /go
ENV GO111MODULE=on
ENV PATH /usr/local/go/bin:/go/bin:$PATH

# Copy the script files from build-tools stage
COPY --from=build-tools /root/*.sh /root/

# Set the default working directory
WORKDIR /go

# Set the default command
CMD ["/bin/bash"]

