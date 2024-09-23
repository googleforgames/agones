# Copyright 2024 Google LLC All Rights Reserved.
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

FROM gcr.io/cloud-builders/gcloud AS builder

RUN apt-get update && \
    apt-get install -y curl && \
    apt-get clean

WORKDIR /usr/local

# install kubectl
ENV KUBECTL_VER=1.29.7
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VER}/bin/linux/amd64/kubectl && \
    chmod go+rx ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

# install Helm package manager
ENV HELM_VER=3.14.3
ENV HELM_URL=https://get.helm.sh/helm-v${HELM_VER}-linux-amd64.tar.gz
RUN curl -L  ${HELM_URL} > /tmp/helm.tar.gz \
    && tar -zxvf /tmp/helm.tar.gz -C /tmp \
    && mv /tmp/linux-amd64/helm /usr/local/bin/helm \
    && chmod go+rx /usr/local/bin/helm \
    && rm /tmp/helm.tar.gz && rm -rf /tmp/linux-amd64

# Build the Go image from source
FROM golang:1.22.6 AS build-stage

WORKDIR /agones.dev

COPY *.go ./

RUN go mod init agones.dev/agones/test/upgrade/testContainer
RUN go mod tidy
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /upgrade-test

# Copy the above binary into a lean image
FROM gcr.io/distroless/static-debian12:nonroot AS build-release-stage

WORKDIR /

COPY --from=build-stage /upgrade-test /upgrade-test
COPY --from=builder /usr/local /usr/local

USER nonroot:nonroot

ENTRYPOINT ["/upgrade-test"]
