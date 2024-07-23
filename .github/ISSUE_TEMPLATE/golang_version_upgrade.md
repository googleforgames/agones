---
name: Golang version upgrade
about: Steps to upgrade Golang version
title: ''
labels: kind/cleanup
assignees: ''

---


Steps to upgrade Golang version:
- [ ] Update `go.mod` and `go.sum`. At the root of the directory, run:
    - [ ] `find . -name 'go.mod' -not -path '*/\.*' -execdir go mod edit -go=<NEW_GOLANG_VERSION_WITHOUT_PATCH> \;`
    - [ ] `find . -name 'go.mod' -not -path '*/\.*' -execdir go mod tidy \;`

- [ ] Update the Dockerfiles for `build` directory. At the root of the directory, run: 
    
    `find build -type f \( -not -path '*/\.*' -and -not -path 'build/tmp/*' \) -exec sed -i 's/GO_VERSION=[0-9]\+\.[0-9]\+\.[0-9]\+/GO_VERSION=<NEW_GOLANG_VERSION>/g' {} \;`
    
- [ ] Update the Dockerfiles for `examples` directory. At the root of the directory, run:     
    - [ ] `find examples -name Dockerfile -exec sed -i 's/golang:[0-9]\+\.[0-9]\+-alpine/golang:<NEW_GOLANG_VERSION_WITHOUT_PATCH>-alpine/g' {} \;`
    - [ ] `find examples \( -name Dockerfile -o -name Dockerfile.windows \) -exec sed -i 's/golang:[0-9]\+\.[0-9]\+\.[0-9]\+/golang:<NEW_GOLANG_VERSION>/g' {} \;`
    
- [ ] Update the example images tag. At `build` directory, run:
    - [ ] `make bump-image IMAGENAME=allocation-endpoint-proxy VERSION=<current-image-version>`
    - [ ] `make bump-image IMAGENAME=autoscaler-webhook VERSION=<current-image-version>`
    - [ ] `make bump-image IMAGENAME=crd-client VERSION=<current-image-version>`
    - [ ] `make bump-image IMAGENAME=custom-controller VERSION=<current-image-version>`
    - [ ] `make bump-image IMAGENAME=simple-game-server VERSION=<current-image-version>`
    - [ ] `make bump-image IMAGENAME=simple-genai-game-server VERSION=<current-image-version>`
    - [ ] `make bump-image IMAGENAME=supertuxkart-example VERSION=<current-image-version>`
    - [ ] `make bump-image IMAGENAME=xonotic-example VERSION=<current-image-version>`

- [ ] Create a PR for the above changes and send for review

- [ ] After the above PR is approved, **before** merging it, run the following to generate and push the new example images:
    - [ ] In `examples/allocation-endpoint`, run: `make cloud-build`
    - [ ] In `examples/autoscaler-webhook`, run: `make cloud-build`
    - [ ] In `examples/crd-client`, run: `make cloud-build`
    - [ ] In `examples/custom-controller`, run: `make cloud-build`
    - [ ] In `examples/simple-game-server`, run: `make cloud-build`
    - [ ] In `examples/simple-genai-server`, run: `make cloud-build`
    - [ ] In `examples/supertuxkart`, run: `make cloud-build`
    - [ ] In `examples/xonotic`, run: `make cloud-build`

- [ ] Merge the above PR