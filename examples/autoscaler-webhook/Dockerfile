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

# Gather dependencies and build the executable
FROM golang:1.19.1 as builder
WORKDIR /go/src/autoscaler-webhook

COPY examples/autoscaler-webhook/main.go .
COPY . /go/src/agones.dev/agones
RUN go mod init && go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

# Create the final image that will run the webhook server for FleetAutoscaler webhook policy
FROM alpine:3.14
RUN adduser -D -u 1000 server

COPY --from=builder /go/src/autoscaler-webhook \
                    /home/server

RUN chown -R server /home/server && \
    chmod o+x /home/server/server

USER 1000
ENTRYPOINT /home/server/server
