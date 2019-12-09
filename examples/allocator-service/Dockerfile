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

# Gather dependencies and build the executable
FROM golang:1.10.3 as builder

WORKDIR /go/src/agones.dev
# TODO: Clone from the latest release branch instead of from the master branch.
RUN git clone https://github.com/googleforgames/agones.git

WORKDIR /go/src/agones.dev/agones/examples/allocator-service
ADD ./main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service .


# Create the final image that will run the allocator service
FROM alpine:3.8
RUN apk add --update ca-certificates
RUN adduser -D service

COPY --from=builder /go/src/agones.dev/agones/examples/allocator-service \
                    /home/service

RUN chown -R service /home/service && \
    chmod o+x /home/service/service

USER service
ENTRYPOINT /home/service/service
