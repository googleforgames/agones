FROM gcr.io/distroless/static-debian12:nonroot

ARG ARCH=amd64
WORKDIR /

COPY ./bin/processor.linux.$ARCH /processor
COPY ./bin/LICENSES .
COPY ./bin/dependencies-src.tgz .

USER nonroot:nonroot
ENTRYPOINT ["/processor"]
