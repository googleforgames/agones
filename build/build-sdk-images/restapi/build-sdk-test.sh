#!/usr/bin/env bash

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

mkdir /go/src/agones.dev/agones/swagger
wget -q https://repo1.maven.org/maven2/io/swagger/swagger-codegen-cli/2.4.10/swagger-codegen-cli-2.4.10.jar -O /tmp/swagger-codegen-cli.jar
java -jar /tmp/swagger-codegen-cli.jar generate -i /go/src/agones.dev/agones/sdks/swagger/sdk.swagger.json  -l go -o /go/src/agones.dev/agones/test/sdk/restapi/swagger
java -jar /tmp/swagger-codegen-cli.jar generate -i /go/src/agones.dev/agones/sdks/swagger/alpha.swagger.json  -l go -o /go/src/agones.dev/agones/test/sdk/restapi/alpha/swagger
