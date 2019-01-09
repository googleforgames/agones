FROM alpine:3.8

RUN apk --update add ca-certificates && \
    adduser -D agones

COPY  --chown=agones:root ./bin/controller /home/agones/controller

USER agones
ENTRYPOINT ["/home/agones/controller"]