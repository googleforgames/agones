FROM alpine:3.7

COPY ./bin/sidecar.linux.amd64 /home/agon/sidecar
RUN apk --update add ca-certificates && \
                adduser -D agon && \
                chown -R agon /home/agon && \
                chmod o+x /home/agon/sidecar

USER agon
ENTRYPOINT ["/home/agon/sidecar"]