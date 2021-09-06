# Copyright 2021 Google LLC All Rights Reserved.
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
FROM golang:1.14.12 as builder

WORKDIR /go/src/minetest

COPY main.go .
RUN go get agones.dev/agones/pkg/sdk \
    && go build -o wrapper .

# final image
FROM debian:buster-backports

RUN set -x \
	&& apt-get update \
    && apt-get install -y --no-install-recommends --no-install-suggests \ 
    -t buster-backports minetest-server/buster-backports \
    && { \
		echo '#!/bin/bash'; \
		echo 'while true'; \
		echo 'do'; \
		echo '    /usr/games/minetestserver --gameid minetest --worldname theWorld --config /etc/minetest/minetest.conf 2>&1'; \
		echo '    sleep 10'; \
		echo 'done'; \
	   } > "/usr/games/minetestserver.sh"

COPY minetest.conf /etc/minetest/minetest.conf
COPY --from=builder /go/src/minetest/wrapper /usr/games/

# Details of setting up a Minetest dedicated server is at https://wiki.minetest.net/Setting_up_a_server/Debian
RUN chown -R Debian-minetest /usr/games/wrapper /usr/games/minetestserver.sh /etc/minetest/minetest.conf 
USER Debian-minetest
RUN chmod +x /usr/games/wrapper \
    && chmod +x /usr/games/minetestserver.sh

ENTRYPOINT ["/usr/games/wrapper", "-i", "/usr/games/minetestserver.sh"]

# Expose ports
EXPOSE 30000/udp