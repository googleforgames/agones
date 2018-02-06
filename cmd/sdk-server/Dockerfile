FROM alpine:3.7

COPY ./bin/sdk-server.linux.amd64 /home/agon/sdk-server
RUN apk --update add ca-certificates && \
                adduser -D agon && \
                chown -R agon /home/agon && \
                chmod o+x /home/agon/sdk-server

USER agon
ENTRYPOINT ["/home/agon/sdk-server"]