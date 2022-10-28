#!/usr/bin/env bash

# Copyright 2022 Google LLC All Rights Reserved.
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

header() {
    cat /go/src/agones.dev/agones/build/boilerplate.go.txt "$1" | sponge "$1"
}

wget -q https://repo1.maven.org/maven2/io/swagger/codegen/v3/swagger-codegen-cli/3.0.35/swagger-codegen-cli-3.0.35.jar -O /tmp/swagger-codegen-cli.jar
java -jar /tmp/swagger-codegen-cli.jar generate -i /go/src/agones.dev/agones/sdks/swagger/sdk.swagger.json  -l go -o /go/src/agones.dev/agones/test/sdk/restapi/swagger
java -jar /tmp/swagger-codegen-cli.jar generate -i /go/src/agones.dev/agones/sdks/swagger/alpha.swagger.json  -l go -o /go/src/agones.dev/agones/test/sdk/restapi/alpha/swagger

# remove un-used files
rm -rf /go/src/agones.dev/agones/test/sdk/restapi/swagger/.*
rm -rf /go/src/agones.dev/agones/test/sdk/restapi/swagger/*.md
rm -rf /go/src/agones.dev/agones/test/sdk/restapi/swagger/*.sh
rm -rf /go/src/agones.dev/agones/test/sdk/restapi/swagger/docs
rm -rf /go/src/agones.dev/agones/test/sdk/restapi/swagger/api
rm -rf /go/src/agones.dev/agones/test/sdk/restapi/alpha/swagger/.*
rm -rf /go/src/agones.dev/agones/test/sdk/restapi/alpha/swagger/*.md
rm -rf /go/src/agones.dev/agones/test/sdk/restapi/alpha/swagger/*.sh
rm -rf /go/src/agones.dev/agones/test/sdk/restapi/alpha/swagger/docs
rm -rf /go/src/agones.dev/agones/test/sdk/restapi/alpha/swagger/api

for file in `ls /go/src/agones.dev/agones/test/sdk/restapi/swagger`
do
  header /go/src/agones.dev/agones/test/sdk/restapi/swagger/${file}
done

for alpha in `ls /go/src/agones.dev/agones/test/sdk/restapi/alpha/swagger`
do
  header /go/src/agones.dev/agones/test/sdk/restapi/alpha/swagger/${alpha}
done
