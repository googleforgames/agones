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
FROM debian:bookworm AS build-tools

# Install dependencies for building the dummy package
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    gnupg \
    curl \
    git \
    wget \
    rsync \
    make \
    python3 \
    bash-completion \
    zip \
    nano \
    jq \
    graphviz \
    gettext-base \
    plantuml \
    software-properties-common \
    ruby-full \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Create the dummy python2.7 package and specify the output location as /opt
RUN gem install fpm && \
    fpm -s empty -t deb -p /opt/python2.7-0.0.0-0-any.deb --name python2.7 --license MIT --version 0.0.0 --architecture all --description 'Dummy package to satisfy google-cloud-cli dependencies.' --maintainer 'Yourself' && \
    dpkg -i /opt/python2.7-0.0.0-0-any.deb && \
    rm /opt/python2.7-0.0.0-0-any.deb

# Install Go
WORKDIR /usr/local
ENV GO_VERSION=1.21.12
RUN wget -q https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz && \
    mkdir /go

ENV GOPATH /go
ENV GO111MODULE=on
ENV PATH /usr/local/go/bin:/go/bin:$PATH

# Install Go tools
RUN go install golang.org/x/tools/cmd/goimports@latest && \
    go install golang.org/x/pkgsite/cmd/pkgsite@latest

# Install golangci-lint
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install Helm
ENV HELM_VER 3.10.3
RUN curl -L https://get.helm.sh/helm-v${HELM_VER}-linux-amd64.tar.gz | tar -xz -C /tmp && \
    mv /tmp/linux-amd64/helm /usr/local/bin/helm && \
    chmod go+rx /usr/local/bin/helm

# Install Hugo
ENV HUGO_VER 0.118.2
RUN wget -q https://github.com/gohugoio/hugo/releases/download/v${HUGO_VER}/hugo_extended_${HUGO_VER}_Linux-64bit.tar.gz -O /tmp/hugo.tar.gz && \
    tar -xzvf /tmp/hugo.tar.gz -C /usr/local/bin/ hugo && rm /tmp/hugo.tar.gz

# Install Node.js (reduced layers)
RUN apt-get update && apt-get install -y ca-certificates curl gnupg && \
    mkdir -p /etc/apt/keyrings && \
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg && \
    echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_20.x nodistro main" | tee /etc/apt/sources.list.d/nodesource.list && \
    apt-get update && apt-get install -y --no-install-recommends nodejs && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Final Stage - Application Runtime
FROM debian:bookworm-slim AS runtime

# Installing only the necessary runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    psmisc \
    rsync \
    bash-completion \
    ca-certificates \
    curl \
    gnupg \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Copy the dummy python2.7 package from the build-tools stage to /tmp/
COPY --from=build-tools /opt/python2.7-0.0.0-0-any.deb /tmp/

# Install dummy python2.7 package
RUN dpkg -i /tmp/python2.7-0.0.0-0-any.deb && rm /tmp/python2.7-0.0.0-0-any.deb

# Copy necessary binaries from build-tools stage
COPY --from=build-tools /usr/local/go /usr/local/go
COPY --from=build-tools /usr/local/bin/helm /usr/local/bin/helm
COPY --from=build-tools /usr/local/bin/hugo /usr/local/bin/hugo
COPY --from=build-tools /go/bin/golangci-lint /usr/local/bin/golangci-lint
COPY --from=build-tools /go/bin /go/bin

ENV GOPATH /go
ENV GO111MODULE=on
ENV PATH /usr/local/go/bin:/go/bin:$PATH

# Install Google Cloud CLI
RUN echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] http://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - && \
    apt-get update && apt-get install -y google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin google-cloud-cli-app-engine-go && \
    echo "source /usr/share/google-cloud-sdk/completion.bash.inc" >> /root/.bashrc

# Install specific kubectl version
ENV KUBERNETES_VER 1.29.7
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VER}/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && mv ./kubectl /usr/local/bin/kubectl

# Copy the script files 
COPY --from=build-tools /root/*.sh /root/

# Final setup
WORKDIR /go
CMD ["/bin/bash"]

