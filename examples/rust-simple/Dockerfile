# Copyright 2018 Google LLC All Rights Reserved.
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

FROM rust:1.41.0 as builder
RUN useradd -m build

# Rust SDK depends on https://github.com/pingcap/grpc-rs and it requires CMake and Go

## Cmake
ENV CMAKE_MINOR_VERSION=v3.10 \
    CMAKE_FULL_VERSION=3.10.3
RUN mkdir -p /usr/src/cmake \
  && curl -fSLO https://cmake.org/files/${CMAKE_MINOR_VERSION}/cmake-${CMAKE_FULL_VERSION}.tar.gz \
  && curl -fSLO https://cmake.org/files/${CMAKE_MINOR_VERSION}/cmake-${CMAKE_FULL_VERSION}-SHA-256.txt \
  && sha256sum -c cmake-${CMAKE_FULL_VERSION}-SHA-256.txt 2>&1 | grep OK \
  && tar xf cmake-${CMAKE_FULL_VERSION}.tar.gz -C /usr/src/cmake --strip-components=1 \
  && rm -f cmake-${CMAKE_FULL_VERSION}.* \
  && cd /usr/src/cmake \
  && ./configure && make -j$(nproc) && make install

## Go
ENV GO_VERSION=1.10.2 \
    GO_CHECKSUM=4b677d698c65370afa33757b6954ade60347aaca310ea92a63ed717d7cb0c2ff
RUN mkdir -p /usr/local/go \
  && curl -fSO https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz \
  && shasum -a 256 go${GO_VERSION}.linux-amd64.tar.gz | grep ${GO_CHECKSUM} \
  && tar xf go${GO_VERSION}.linux-amd64.tar.gz -C /usr/local/go --strip-components=1 \
  && rm -f go${GO_VERSION}.linux-amd64.tar.gz
ENV PATH $PATH:/usr/local/go/bin

# SDK
COPY sdk/src /home/builder/agones/sdks/rust/src
COPY sdk/Cargo.toml /home/builder/agones/sdks/rust/

# Example
COPY src /home/builder/agones/examples/rust-simple/src
COPY Cargo.toml /home/builder/agones/examples/rust-simple/
COPY Makefile /home/builder/agones/examples/rust-simple/

WORKDIR /home/builder/agones/examples/rust-simple
RUN make build-server

FROM debian:buster
RUN useradd -u 1000 -m server

COPY --from=builder --chown=server:server /home/builder/agones/examples/rust-simple/target/release/rust-simple /home/server/rust-simple

USER 1000
ENTRYPOINT /home/server/rust-simple
