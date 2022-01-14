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

COPY ./examples/nodejs-simple/node-v12.22.9-linux-x64.tar.gz .
RUN tar xvf node-v12.22.9-linux-x64.tar.gz
RUN cd node-v12.22.9-linux-x64 && ln -s $(pwd)/bin/node /usr/bin/node && ln -s $(pwd)/bin/npm /usr/bin/npm

# Installing request is to address a bug with node-pre-gyp
RUN npm install --unsafe-perm --global request@2.88.2 grpc-tools@1.8.1

# code generation scripts
COPY scripts/node/*.sh /root/
RUN chmod +x /root/*.sh