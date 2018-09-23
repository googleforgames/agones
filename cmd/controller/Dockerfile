FROM alpine:3.8

RUN apk --update add ca-certificates && \
    adduser -D agones

COPY ./bin/controller /home/agones/controller

RUN chown -R agones /home/agones && \
    chmod o+x /home/agones/controller

USER agones
ENTRYPOINT ["/home/agones/controller"]