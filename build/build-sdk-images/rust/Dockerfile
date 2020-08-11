# Copyright 2019 Google LLC All Rights Reserved.
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
ARG BASE_IMAGE=agones-build-sdk-base:latest
FROM $BASE_IMAGE

RUN apt-get update && \
    apt-get install -y wget && \
    apt-get clean

# install rust
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH \
    RUST_VERSION=1.45.0
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
RUN cargo install protobuf-codegen --vers 2.16.2
RUN cargo install grpcio-compiler --vers 0.6.0

RUN wget -q https://cmake.org/files/v3.14/cmake-3.14.1-Linux-x86_64.sh
RUN mkdir /opt/cmake
RUN sh cmake-3.14.1-Linux-x86_64.sh --prefix=/opt/cmake --skip-license
RUN ln -s /opt/cmake/bin/cmake /usr/local/bin/cmake
RUN cmake --version

ENV GO_VERSION=1.10.2 \
    GO_CHECKSUM=4b677d698c65370afa33757b6954ade60347aaca310ea92a63ed717d7cb0c2ff
RUN mkdir -p /usr/local/go \
  && curl -fSO https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz \
  && shasum -a 256 go${GO_VERSION}.linux-amd64.tar.gz | grep ${GO_CHECKSUM} \
  && tar xf go${GO_VERSION}.linux-amd64.tar.gz -C /usr/local/go --strip-components=1 \
  && rm -f go${GO_VERSION}.linux-amd64.tar.gz
ENV PATH $PATH:/usr/local/go/bin

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh