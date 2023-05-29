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

ARG USER_ID
ARG GROUP_ID
RUN userdel -f agones-dev && if getent group agones-dev ; then groupdel agones-dev; fi &&\
    groupadd -g ${GROUP_ID:-1338} agones-dev && \
    useradd -l -u ${USER_ID:-1337} -g agones-dev agones-dev

RUN apt-get --allow-releaseinfo-change update && \
    apt-get install -y wget && \
    apt-get clean

# install rust
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:$PATH \
    RUST_VERSION=1.61.0
RUN wget -q https://static.rust-lang.org/rustup/dist/x86_64-unknown-linux-gnu/rustup-init && \
    chmod +x rustup-init && \
    ./rustup-init -y --no-modify-path --default-toolchain $RUST_VERSION && \
    rm rustup-init && \
    rustup --version; \
    cargo --version; \
    rustc --version;

WORKDIR /go/src/agones.dev/agones

# code generation scripts
COPY *.sh /home/agones-dev/
RUN chmod +x /home/agones-dev/*.sh

RUN chown --silent --no-dereference --recursive ${USER_ID:-1337}:${GROUP_ID:-1338} /home/agones-dev /go
USER agones-dev
