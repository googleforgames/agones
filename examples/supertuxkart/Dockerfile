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

# build go
FROM golang:1.17.2 as go-builder
WORKDIR /go/src/supertuxkart

RUN apt-get update && apt-get install -y curl tar xz-utils

COPY main.go .
COPY go.mod .
RUN go mod tidy
RUN go build -o wrapper .

# compile stk
FROM debian:buster as stk-builder

RUN apt update && apt-get install -y build-essential cmake libbluetooth-dev \
libcurl4-openssl-dev libenet-dev libfreetype6-dev libharfbuzz-dev libfribidi-dev \
libgl1-mesa-dev libglew-dev libjpeg-dev libogg-dev libopenal-dev libpng-dev \
libssl-dev libvorbis-dev libxrandr-dev libx11-dev nettle-dev pkg-config zlib1g-dev \
git subversion wget xz-utils

RUN wget https://github.com/supertuxkart/stk-code/releases/download/1.3/SuperTuxKart-1.3-linux-64bit.tar.xz

RUN tar -xf SuperTuxKart-1.3-linux-64bit.tar.xz
RUN rm -rf SuperTuxKart-1.3-linux-64bit.tar.xz
RUN ./SuperTuxKart-1.3-linux-64bit/run_game.sh

# final image
FROM debian:bullseye

RUN apt update && apt-get install -y libbluetooth3 libbz2-1.0 libc6 \
 libcurl4 libenet7 libfreetype6 libfribidi0 libgcc1 libgl1 libglu1-mesa \
 libglu1 libjpeg62-turbo libogg0 libopenal1 libpng16-16 libstdc++6 libvorbis0a \
 libvorbisenc2 libvorbisfile3 libx11-6 libxrandr2 zlib1g libglew2.1 libharfbuzz0b

RUN useradd -m  -u 1000 supertuxkart
WORKDIR /home/supertuxkart

WORKDIR /home/supertuxkart/stk-code

COPY --from=go-builder /go/src/supertuxkart/wrapper .
COPY entrypoint.sh .
COPY server_config.xml .

RUN chown -R supertuxkart:supertuxkart /home/supertuxkart && chmod +x wrapper

ENV ENABLE_PLAYER_TRACKING=false
USER 1000
ENTRYPOINT ["./entrypoint.sh"]