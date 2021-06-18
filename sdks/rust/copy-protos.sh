#!/usr/bin/env bash
set -e

dir=$(dirname "$0")

rm -rf "${dir}/proto"

cp -r "${dir}/../../proto" "${dir}"
