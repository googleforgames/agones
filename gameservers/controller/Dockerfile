FROM alpine:3.6

COPY ./bin/controller /home/agon/controller
RUN apk --update add ca-certificates && \
                adduser -D agon && \
                chown -R agon /home/agon && \
                chmod o+x /home/agon/controller

USER agon
ENTRYPOINT ["/home/agon/controller"]