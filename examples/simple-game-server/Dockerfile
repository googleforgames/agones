# Copyright 2020 Google LLC All Rights Reserved.
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

# build
FROM golang:1.16.5 as builder
WORKDIR /go/src/simple-game-server

COPY examples/simple-game-server/main.go .
COPY . /go/src/agones.dev/agones
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .

# final image
FROM alpine:3.11

RUN adduser -D -u 1000 server
COPY --from=builder /go/src/simple-game-server/server /home/server/server
RUN chown -R server /home/server && \
    chmod o+x /home/server/server

USER 1000
ENTRYPOINT ["/home/server/server"]
