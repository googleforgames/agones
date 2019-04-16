#!/bin/bash

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

cd /go/src/github.com/ahmetb/gen-crd-api-reference-docs

#Use local version of agones
go mod edit --replace=agones.dev/agones@latest=../../../agones.dev/agones/
go build

cp /go/src/agones.dev/agones/site/assets/templates/pkg.tpl ./template

FILE=${FILE:-/go/src/agones.dev/agones/site/content/en/docs/Reference/agones_crd_api_reference.html}
VERSION=${VERSION:-"0.9.0"}

HEAD="/tmp/head.html"
RESULT="/tmp/agones_crd_api_reference.html"
OLD="/tmp/old_docs.html"

./gen-crd-api-reference-docs --config ./example-config.json --api-dir ../../../agones.dev/agones/pkg/apis/stable/v1alpha1/ --out-file $RESULT
awk '/\ feature\ publishVersion/{flag=1;next}/\ \/feature/{flag=0}flag' $FILE > $OLD

awk '//{flag=1}/\ feature\ publishVersion/{flag=0;exit}flag' $FILE > $HEAD
doc_version=$(grep 'feature publishVersion=' $FILE )
echo $doc_version
publish='{{% feature publishVersion="'$VERSION'" %}}'
expiry='{{% feature expiryVersion="'$VERSION'" %}}'
function sedeasy {
  sed -i "s/$(echo $1 | sed -e 's/\([[\/.*]\|\]\)/\\&/g')/$(echo $2 | sed -e 's/[\/&]/\\&/g')/g" $3
}

diff $RESULT $OLD
if [ $? -gt 0 ]
then 
echo "Output to a file $FILE"

if [ "$publish" != "$doc_version" ]
then 
    echo "Tagging old docs with expiryVersion shortcode"

    sedeasy "$doc_version" "$expiry"  $FILE 
    cat $FILE > $HEAD
fi
cat $HEAD > $FILE
echo -e '\n{{% feature publishVersion="'$VERSION'" %}}' >> $FILE
cat $RESULT >> $FILE
echo -e '{{% /feature %}}\n' >> $FILE
fi
