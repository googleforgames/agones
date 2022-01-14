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

FROM debian:buster
RUN useradd -u 1000 -m server

WORKDIR /home/server/examples/nodejs-simple
COPY /examples/nodejs-simple .
RUN tar xvf node-v12.22.9-linux-x64.tar.gz
RUN cd node-v12.22.9-linux-x64 && ln -s $(pwd)/bin/node /usr/bin/node && ln -s $(pwd)/bin/npm /usr/bin/npm

RUN npm install && npm rebuild
RUN chown -R server /home/server
USER 1000

ENTRYPOINT ["npm", "start", "--"]
