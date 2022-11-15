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

FROM debian:bullseye
RUN useradd -u 1000 -m server
RUN apt-get update && apt-get install -y curl software-properties-common gnupg  && \
    apt-get clean

RUN add-apt-repository -y -r ppa:chris-lea/node.js
RUN rm -f /etc/apt/sources.list.d/chris-lea-node_js-*.list
RUN rm -f /etc/apt/sources.list.d/chris-lea-node_js-*.list.save

ARG KEYRING=/usr/share/keyrings/nodesource.gpg
ARG VERSION=node_18.x

RUN curl -fsSL https://deb.nodesource.com/gpgkey/nodesource.gpg.key | gpg --dearmor | tee "$KEYRING" >/dev/null
RUN gpg --no-default-keyring --keyring "$KEYRING" --list-keys

ARG DISTRO="bullseye"
RUN echo "deb [signed-by=$KEYRING] https://deb.nodesource.com/$VERSION $DISTRO main" | tee /etc/apt/sources.list.d/nodesource.list
RUN echo "deb-src [signed-by=$KEYRING] https://deb.nodesource.com/$VERSION $DISTRO main" | tee -a /etc/apt/sources.list.d/nodesource.list

RUN apt-get update && apt-get install -y nodejs

WORKDIR /home/server

COPY ./sdks/nodejs sdks/nodejs
RUN cd sdks/nodejs && \
    npm ci --production
COPY ./examples/nodejs-simple examples/nodejs-simple
RUN cd examples/nodejs-simple && \
    npm ci --production
RUN chown -R server /home/server
USER 1000

WORKDIR /home/server/examples/nodejs-simple
ENTRYPOINT ["npm", "start", "--"]
