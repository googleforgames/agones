# Copyright 2017 Google Inc. All Rights Reserved.
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

# ForceUpdate 6 -- change here if you need to force a rebuild

# compiling proto + grpc takes an exceptionally long time
# so we'll use a base from `base` - which is manually built using the below tag.
FROM gcr.io/agones-images/grpc-cxx:1.12

RUN apt-get update && \
    apt-get install -y wget psmisc rsync make python bash-completion zip nano jq && \
    apt-get clean

# install go
WORKDIR /usr/local
ENV GO_VERSION=1.10.3
ENV GOPATH /go
RUN wget -q https://redirector.gvt1.com/edgedl/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -xzf go${GO_VERSION}.linux-amd64.tar.gz && rm go${GO_VERSION}.linux-amd64.tar.gz && mkdir ${GOPATH}

# install gcloud + kubectl, because it's an easy way to test/dev against kubernetes.
WORKDIR /opt
RUN wget -q https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.zip && unzip -q google-cloud-sdk.zip && \
    rm google-cloud-sdk.zip && \
    /opt/google-cloud-sdk/install.sh --usage-reporting=true --path-update=true --bash-completion=true --rc-path=/root/.bashrc

# update the path for both go and gcloud
ENV PATH /usr/local/go/bin:/go/bin:/opt/google-cloud-sdk/bin:$PATH

# RUN gcloud components update
RUN gcloud components update

# install kubectl without gcloud as we want the last version
ENV KUBECTL_VER 1.10.0
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VER}/bin/linux/amd64/kubectl && \
    chmod go+rx ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl
RUN echo "source <(kubectl completion bash)" >> /root/.bashrc

# install Helm package manager
ENV HELM_VER 2.9.1
ENV HELM_URL https://storage.googleapis.com/kubernetes-helm/helm-v${HELM_VER}-linux-amd64.tar.gz
RUN curl -L  ${HELM_URL} > /tmp/helm.tar.gz \
    && tar -zxvf /tmp/helm.tar.gz -C /tmp \
    && mv /tmp/linux-amd64/helm /usr/local/bin/helm \
    && chmod go+rx /usr/local/bin/helm \
    && rm /tmp/helm.tar.gz && rm -rf /tmp/linux-amd64
RUN echo "source <(helm completion bash)" >> /root/.bashrc

# install go-proto-gen 1.1
RUN mkdir -p /go/src/github.com/golang && cd /go/src/github.com/golang && \
    git clone https://github.com/golang/protobuf.git &&  \
    cd protobuf && git checkout v1.1.0 && \
    go install github.com/golang/protobuf/protoc-gen-go

# install go tooling for development, building and testing
RUN go get -u github.com/golang/dep/cmd/dep && \
    go get -u golang.org/x/tools/cmd/goimports

# install golang-ci linter
RUN curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | \
    bash -s -- -b $GOPATH/bin v1.10.2

# install the release branch of the code generator tools
RUN mkdir -p /go/src && cd /go/src && mkdir -p k8s.io && cd k8s.io && \
    git clone -b kubernetes-1.10.5 --depth=3 https://github.com/kubernetes/code-generator.git

# install rust
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH \
    RUST_VERSION=1.26.1
ENV RUST_ARCH=x86_64-unknown-linux-gnu \
    RUSTUP_SHA256=c9837990bce0faab4f6f52604311a19bb8d2cde989bea6a7b605c8e526db6f02
RUN wget -q https://static.rust-lang.org/rustup/archive/1.11.0/${RUST_ARCH}/rustup-init && \
    echo "${RUSTUP_SHA256} *rustup-init" | sha256sum -c - && \
    chmod +x rustup-init && \
    ./rustup-init -y --no-modify-path --default-toolchain $RUST_VERSION && \
    rm rustup-init && \
    rustup --version; \
    cargo --version; \
    rustc --version;

# install rust tooling for SDK generation
RUN cargo install protobuf-codegen --vers 2.0.2
RUN cargo install grpcio-compiler --vers 0.3.0

# make sure we keep the path to go
RUN echo "export PATH=/usr/local/go/bin:/go/bin/:\$PATH" >> /root/.bashrc
# make nano the editor
RUN echo "export EDITOR=nano" >> /root/.bashrc

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh

WORKDIR /go