#!/usr/bin/env bash

#
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
#

set -o errexit
set -o nounset
set -o pipefail

echo
echo "⚠ If you run into errors running this script, please remove anything you have installed in your cluster, or create a brand new cluster ⚠"
echo

do_expand() {
  echo "Processing $1"
  jq '.definitions."'"$1"'"' ./openapi.json >"$1.json"
  children=$(jq -r '..|objects|.["$ref"]|select (.!=null)' "$1.json" | sed 's!#/definitions/!!')
  if [ -n "$children" ]; then
    while IFS= read -r line; do
      do_expand "$line"
    done <<<"$children"
  fi
}

json_2_helm_yaml() {
  yaml="_$1.yaml"
  echo "Converting to YAML: $1"
  ./yq eval -P "$1.json" >"$yaml"

  echo "{{- define \"$1\" }}" | cat ../boilerplate.yaml.txt - "$yaml" | sponge "$yaml"
  echo "{{- end }}" >>"$yaml"
  mv "$yaml" ../../install/helm/agones/templates/crds/k8s/
}

rm -r ./tmp || true
mkdir tmp
cd tmp
kubectl proxy &

# install deps while waiting for kubectl proxy to startup
wget https://github.com/mikefarah/yq/releases/download/v4.3.2/yq_linux_amd64 -O yq
apt install -y moreutils
chmod +x ./yq

# cleanup and format
curl http://127.0.0.1:8001/openapi/v2 | jq 'del(.. | .["x-kubernetes-patch-strategy"]?, .["x-kubernetes-patch-merge-key"]?, .["x-kubernetes-list-type"]?)' |
  jq 'del(.. |  .["x-kubernetes-group-version-kind"]?, .["x-kubernetes-list-map-keys"]?, .["x-kubernetes-unions"]? )' >openapi.json
do_expand "io.k8s.api.core.v1.PodTemplateSpec"

# do any editing you need to do to any types here, before they are expanded.
# creationTimestamp is defaulted to it's zero value, which gets serialised as "null", so we have to set it as nullable.
jq '.properties.creationTimestamp.nullable = true' io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta.json | sponge io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta.json
# Make IntOrString type have `x-kubernetes-int-or-string` member
jq '.["x-kubernetes-int-or-string"] = true' io.k8s.apimachinery.pkg.util.intstr.IntOrString.json | jq 'del(.type)' | sponge io.k8s.apimachinery.pkg.util.intstr.IntOrString.json
# This has a \\ in a description field that is being extra tricky to resolve, so let's just escape it here.
sed -i 's/\\\\/\\\\\\\\/g' io.k8s.api.core.v1.PodSpec.json

# easier debugging if something goes wrong, still have the original
mkdir orig
cp *.json orig
rm openapi.json

for f in *.json; do
  echo "Expanding $f"
  # remove description, because there is usually another description when at its replacement point
  # replace \" with \\" as it get re-unescaped somewhere in the pipe
  # any "foo\nbar" values need their \n escaped
  # remove top and bottom line to get rid of first { and last } (we know all are formatted because jq)
  # then format for multiline - replace real multilines breaks with \n
  # and escape bash and sed special characters, such as $ and &.
  contents=$(cat "$f" | jq 'del(.description)' | sed 's/\\n/\\\\n/g' | sed '$d' | sed '1d' | sed ':a;N;$!ba;s/\n/\\n/g' | sed 's/\$/\\$/g' | sed 's/\&/\\&/g' | sed 's@\"@\\"@g')
  ref=$(basename "$f" .json)

  find -maxdepth 1 -name '*.json' | xargs sed -i 's@"$ref": "#/definitions/'"$ref"'"@'"$contents"'@g'
done

# convert the ones you want to include via helm to yaml
json_2_helm_yaml "io.k8s.api.core.v1.PodTemplateSpec"
json_2_helm_yaml "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"
