FROM alpine:3.7

COPY ./bin/sdk-server.linux.amd64 /home/agones/sdk-server
RUN apk --update add ca-certificates && \
                adduser -D agones && \
                chown -R agones /home/agones && \
                chmod o+x /home/agones/sdk-server

USER agones
ENTRYPOINT ["/home/agones/sdk-server"]