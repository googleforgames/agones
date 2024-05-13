Steps to upgrade Golang version:
- [ ] At the root of the directory, run:

    `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`
- [ ] Update the Dockerfiles for `build` directory: 
    At the root of the directory, run: 
    
    `find build -type f \( -not -path '*/\.*' -and -not -path 'build/tmp/*' \) -exec sed -i 's/GO_VERSION=[0-9]\+\.[0-9]\+\.[0-9]\+/GO_VERSION=<NEW_GOLANG_VERSION>/g' {} \;`
- [ ] Update `go.mod` files for the `build` directory:
    Look for the `go.mod` files in the sub-directories of `build/script`, for each of them, run:

    `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`
- [ ] Update the Go version for each example that was written in Golang:
    - [ ] Update Dockerfiles for `examples` directory:

        At the `examples` directory, run:
        
        `find examples -name Dockerfile -exec sed -i 's/golang:[0-9]\+\.[0-9]\+-alpine/golang:<NEW_GOLANG_VERSION_WITHOUT_PATCH>-alpine/g' {} \;`

        At the `examples` directory, run:
        
        `find examples -name Dockerfile -o -name Dockerfile.windows -exec sed -i 's/golang:[0-9]\+\.[0-9]\+\.[0-9]\+/golang:<NEW_GOLANG_VERSION>/g' {} \;`
    - [ ] Update `go.mod` files for `examples` directory and update the example image tag:
        - [ ] In `examples/allocation-endpoint/client`, run:
        
            `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`

        - [ ] In `examples/allocation-endpoint/server`, run:

            `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`

        - [ ] In `examples/allocation-webhook`:
            - [ ] Run `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`
            - [ ] In `examples/allocation-webhook/Makefile`, increase the version number by 1 for `autoscaler-webhook`

        - [ ] In `examples/crd-client`:
            - [ ] Run `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`
            - [ ] In `examples/crd-client/Makefile`, increase the version number by 1 for `crd-client`

        - [ ] In `examples/custom-controller`
            - [ ] Run `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`
            - [ ] In `examples/custom-controller/Makefile`, increase the version number by 1 for `custom-controller`

        - [ ] In `examples/simple-game-server`
            - [ ] Run `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`
            - [ ] In `examples/simple-game-server/Makefile`, increase the version number by 1 for `simple-game-server`

        - [ ] In `examples/simple-genai-server`
            - [ ] Run `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`
            - [ ] In `examples/simple-genai-server/Makefile`, increase the version number by 1 for `simple-genai-game-server`

        - [ ] In `examples/supertuxkart`
            - [ ] Run `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`
            - [ ] In `examples/supertuxkart/Makefile`, increase the version number by 1 for `supertuxkart-example`

        - [ ] In `examples/xonotic`
            - [ ] Run `go mod edit -go=<NEW_GOLANG_VERSION> && go mod tidy`
            - [ ] In `examples/xonotic/Makefile`, increase the version number by 1 for `xonotic-example`

- [ ] Create a PR for the above changes and send for review
- [ ] After the above PR is approved, merge it and run the following in `build` directory to generate and push the new example images:

    `export REGISTRY=us-docker.pkg.dev/agones-images/examples`

    `make gcloud-auth-docker`

    `make build-go-examples && make push-example-golang-images`

- [ ] After the new example images have been pushed, update the version of example images where they are used:

    - [ ] Run the following in the root of directory:

        - [ ] `find examples -type f -not -path '*/\.*' -exec sed -i 's@image: us-docker.pkg.dev/agones-images/examples/autoscaler-webhook:[0-9]\+\.[0-9]\+@image: us-docker.pkg.dev/agones-images/examples/autoscaler-webhook:<NEW_IMAGE_VERSION>@g' {} \;`

        - [ ] `find examples -type f -not -path '*/\.*' -exec sed -i 's@image: us-docker.pkg.dev/agones-images/examples/crd-client:[0-9]\+\.[0-9]\+@image: us-docker.pkg.dev/agones-images/examples/crd-client:<NEW_IMAGE_VERSION>@g' {} \;`

        - [ ] `find examples -type f -not -path '*/\.*' -exec sed -i 's@image: us-docker.pkg.dev/agones-images/examples/custom-controller:[0-9]\+\.[0-9]\+@image: us-docker.pkg.dev/agones-images/examples/custom-controller:<NEW_IMAGE_VERSION>@g' {} \;`

        - [ ] `find examples -type f -not -path '*/\.*' -exec sed -i 's@image: us-docker.pkg.dev/agones-images/examples/simple-game-server:[0-9]\+\.[0-9]\+@image: us-docker.pkg.dev/agones-images/examples/simple-game-server:<NEW_IMAGE_VERSION>@g' {} \;`

        - [ ] `find examples -type f -not -path '*/\.*' -exec sed -i 's@image: us-docker.pkg.dev/agones-images/examples/simple-genai-game-server:[0-9]\+\.[0-9]\+@image: us-docker.pkg.dev/agones-images/examples/simple-genai-game-server:<NEW_IMAGE_VERSION>@g' {} \;`

        - [ ] `find examples -type f -not -path '*/\.*' -exec sed -i 's@image: us-docker.pkg.dev/agones-images/examples/supertuxkart-example:[0-9]\+\.[0-9]\+@image: us-docker.pkg.dev/agones-images/examples/supertuxkart-example:<NEW_IMAGE_VERSION>@g' {} \;`

        - [ ] `find examples -type f -not -path '*/\.*' -exec sed -i 's@image: us-docker.pkg.dev/agones-images/examples/xonotic-example:[0-9]\+\.[0-9]\+@image: us-docker.pkg.dev/agones-images/examples/xonotic-example:<NEW_IMAGE_VERSION>@g' {} \;`
        
    - [ ] Create a PR for the above change, and merge the change after it's approved
