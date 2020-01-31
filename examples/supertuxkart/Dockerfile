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
FROM golang:1.13.5 as builder
WORKDIR /go/src/supertuxkart

RUN apt-get update && apt-get install -y curl tar xz-utils

COPY examples/supertuxkart/main.go .
COPY . /go/src/agones.dev/agones
RUN go build -o wrapper .

RUN curl -o supertuxkart.tar.xz https://pilotfiber.dl.sourceforge.net/project/supertuxkart/SuperTuxKart/1.1/SuperTuxKart-1.1-linux.tar.xz && \
  tar -xf supertuxkart.tar.xz

# final image
FROM debian:buster

RUN apt update && apt-get install -y libbluetooth3 libbz2-1.0 libc6 \
 libcurl3-gnutls libenet7 libfreetype6 libfribidi0 libgcc1 libgl1 libglu1-mesa \
 libglu1 libjpeg62-turbo libogg0 libopenal1 libpng16-16 libstdc++6 libvorbis0a \
 libvorbisenc2 libvorbisfile3 libx11-6 libxrandr2 zlib1g

RUN useradd -m  -u 1000 supertuxkart
WORKDIR /home/supertuxkart

COPY --from=builder /go/src/supertuxkart/wrapper .
COPY --from=builder /go/src/supertuxkart/SuperTuxKart-1.1-linux ./supertuxkart
COPY examples/supertuxkart/entrypoint.sh .
COPY examples/supertuxkart/server_config.xml .

RUN chown -R supertuxkart:supertuxkart . && chmod +x wrapper

USER 1000
ENTRYPOINT ["./entrypoint.sh"]
