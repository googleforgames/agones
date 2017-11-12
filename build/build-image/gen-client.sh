#!/usr/bin/env bash
rsync -r /go/src/github.com/agonio/agon/vendor/k8s.io/ /go/src/k8s.io/
/go/src/k8s.io/code-generator/generate-groups.sh "all" \
    github.com/agonio/agon/pkg/client \
    github.com/agonio/agon/pkg/apis stable:v1alpha1 \
    --go-header-file=/go/src/github.com/agonio/agon/build/crd.boilerplate.go.txt
