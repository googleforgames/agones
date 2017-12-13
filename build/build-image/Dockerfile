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

FROM golang:1.9.2

RUN apt-get update && apt-get install -y wget unzip bash-completion rsync

WORKDIR /

# install gcloud + kubectl, because it's an easy way to test/dev against kubernetes.
RUN wget -q https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.zip && unzip -q google-cloud-sdk.zip && rm google-cloud-sdk.zip
RUN /google-cloud-sdk/install.sh --usage-reporting=true --path-update=true --bash-completion=true --rc-path=/root/.bashrc
ENV PATH /google-cloud-sdk/bin:$PATH

RUN gcloud components update && gcloud components install kubectl
RUN echo "source <(kubectl completion bash)" >> /root/.bashrc

# install protoc for grpc
ENV PB_VER 3.5.0
ENV PB_URL https://github.com/google/protobuf/releases/download/v${PB_VER}/protoc-${PB_VER}-linux-x86_64.zip
RUN mkdir -p /tmp/protoc && \
    curl -L ${PB_URL} > /tmp/protoc/protoc.zip && \
    cd /tmp/protoc && \
    unzip protoc.zip && \
    cp /tmp/protoc/bin/protoc /usr/local/bin && \
    cp -R /tmp/protoc/include/* /usr/local/include && \
    chmod go+rx /usr/local/bin/protoc && \
    cd /tmp && \
    rm -r /tmp/protoc

# install go tooling for building and testing
RUN go get -u github.com/golang/dep/cmd/dep && \
    go get -u github.com/alecthomas/gometalinter && \
    go get -u k8s.io/code-generator/... && \
    go get -u github.com/golang/protobuf/protoc-gen-go && \
    /go/bin/gometalinter --install

# make sure we keep the path to go
RUN echo "export PATH=/usr/local/go/bin:/go/bin/:\$PATH" >> /root/.bashrc

# script for generating the CRD scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh

ENTRYPOINT kubectl