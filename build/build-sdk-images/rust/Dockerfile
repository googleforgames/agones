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

# code generation scripts
COPY *.sh /root/
RUN chmod +x /root/*.sh