FROM alpine:3.6

COPY ./bin/sidecar /home/agon/sidecar
RUN apk --update add ca-certificates && \
                adduser -D agon && \
                chown -R agon /home/agon && \
                chmod o+x /home/agon/sidecar

USER agon
ENTRYPOINT /home/agon/sidecar