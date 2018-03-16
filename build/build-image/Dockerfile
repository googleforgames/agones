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

# ForceUpdate 2 -- change here if you need to force a rebuild

# compiling proto + grpc takes an exceptionally long time
# so we'll use that as the base.
FROM grpc/cxx:1.8

RUN apt-get update && \
    apt-get install -y wget rsync make python bash-completion zip && \
    apt-get clean

# install go
WORKDIR /usr/local
ENV GO_VERSION=1.10
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
RUN gcloud components update && gcloud components install kubectl
RUN echo "source <(kubectl completion bash)" >> /root/.bashrc

# install Helm package manager
ENV HELM_VER 2.8.0
ENV HELM_URL https://storage.googleapis.com/kubernetes-helm/helm-v${HELM_VER}-linux-amd64.tar.gz
RUN curl -L  ${HELM_URL} > /tmp/helm.tar.gz \
  && tar -zxvf /tmp/helm.tar.gz -C /tmp \
  && mv /tmp/linux-amd64/helm /usr/local/bin/helm \
  && chmod go+rx /usr/local/bin/helm \
  && rm /tmp/helm.tar.gz && rm -rf /tmp/linux-amd64
RUN echo "source <(helm completion bash)" >> /root/.bashrc

# install go tooling for development, building and testing
RUN go get -u github.com/golang/dep/cmd/dep && \
    go get -u github.com/alecthomas/gometalinter && \
    go get -u github.com/golang/protobuf/protoc-gen-go && \
    /go/bin/gometalinter --install

# install the release branch of the code generator tools
RUN mkdir -p /go/src && cd /go/src && mkdir -p k8s.io && cd k8s.io && \
    git clone -b kubernetes-1.9.2 --depth=3 https://github.com/kubernetes/code-generator.git

# make sure we keep the path to go
RUN echo "export PATH=/usr/local/go/bin:/go/bin/:\$PATH" >> /root/.bashrc

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh

WORKDIR /go