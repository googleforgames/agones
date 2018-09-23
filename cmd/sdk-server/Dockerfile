FROM alpine:3.8

RUN apk --update add ca-certificates && \
    adduser -D agones

COPY ./bin/sdk-server.linux.amd64 /home/agones/sdk-server

RUN chown -R agones /home/agones && \
    chmod o+x /home/agones/sdk-server

USER agones
ENTRYPOINT ["/home/agones/sdk-server"]