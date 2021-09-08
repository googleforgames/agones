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

ARG MINETEST_BUILDER_IMAGE=alpine:3.14
ARG WRAPPER_BUILDER_IMAGE=golang:1.17-alpine3.14
ARG RUNTIME_IMAGE=$MINETEST_BUILDER_IMAGE

FROM $MINETEST_BUILDER_IMAGE AS minetest_builder

WORKDIR /usr/src/minetest

ARG MINETEST_GAME_VERSION=5.4.1
RUN apk add --no-cache git build-base irrlicht-dev cmake bzip2-dev libpng-dev \
		jpeg-dev libxxf86vm-dev mesa-dev sqlite-dev libogg-dev \
		libvorbis-dev openal-soft-dev curl-dev freetype-dev zlib-dev \
		gmp-dev jsoncpp-dev postgresql-dev luajit-dev ca-certificates curl && \
        curl -o minetest-5.4.1.tar.gz -L https://github.com/minetest/minetest/archive/refs/tags/${MINETEST_GAME_VERSION}.tar.gz && \
        tar --strip-components=1 -xzf minetest-5.4.1.tar.gz

WORKDIR /usr/src/
RUN git clone --recursive https://github.com/jupp0r/prometheus-cpp/ && \
	mkdir prometheus-cpp/build && \
	cd prometheus-cpp/build && \
	cmake .. \
		-DCMAKE_INSTALL_PREFIX=/usr/local \
		-DCMAKE_BUILD_TYPE=Release \
		-DENABLE_TESTING=0 && \
	make -j2 && \
	make install

WORKDIR /usr/src/minetest/build
RUN cmake .. \
		-DCMAKE_INSTALL_PREFIX=/usr/local \
		-DCMAKE_BUILD_TYPE=Release \
		-DBUILD_SERVER=TRUE \
		-DENABLE_PROMETHEUS=TRUE \
		-DBUILD_UNITTESTS=FALSE \
		-DBUILD_CLIENT=FALSE && \
	make -j2 && \
	make install

FROM $WRAPPER_BUILDER_IMAGE AS wrapper_builder

WORKDIR /go/src/minetest

COPY main.go go.mod ./
RUN go mod download agones.dev/agones && \
    go mod tidy && \
    go build -o wrapper

FROM $RUNTIME_IMAGE AS runtime

RUN apk add --no-cache sqlite-libs curl gmp libstdc++ libgcc libpq luajit && \
	adduser -D minetest --uid 30000 -h /var/lib/minetest && \
	chown -R minetest:minetest /var/lib/minetest && \
    mkdir -p /usr/local/share/minetest/worlds/devtest

WORKDIR /var/lib/minetest

COPY --from=minetest_builder /usr/local/share/minetest /usr/local/share/minetest
COPY --from=minetest_builder /usr/local/bin/minetestserver /usr/local/bin/minetestserver
COPY --from=wrapper_builder /go/src/minetest/wrapper /usr/local/bin/wrapper
COPY minetest.conf /etc/minetest/minetest.conf
COPY minetestserver.sh /var/lib/minetest/minetestserver.sh

RUN chown -R minetest:minetest /usr/local/bin/wrapper /usr/local/share/minetest \
    /usr/local/bin/minetestserver /var/lib/minetest/minetestserver.sh /etc/minetest/minetest.conf 
USER minetest:minetest
RUN chmod +x /usr/local/bin/wrapper \
    && chmod +x /var/lib/minetest/minetestserver.sh

ENTRYPOINT ["/usr/local/bin/wrapper", "-i", "/var/lib/minetest/minetestserver.sh"]

# Expose ports
EXPOSE 30000/udp
