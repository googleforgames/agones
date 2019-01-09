FROM alpine:3.8

RUN apk --update add ca-certificates && \
    adduser -D agones

COPY --chown=agones:root ./bin/sdk-server.linux.amd64 /home/agones/sdk-server

USER agones
ENTRYPOINT ["/home/agones/sdk-server"]