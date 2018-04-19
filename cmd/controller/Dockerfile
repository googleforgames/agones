FROM alpine:3.7

RUN apk --update add ca-certificates && \
    adduser -D agones

COPY ./certs /home/agones/certs
COPY ./bin/controller /home/agones/controller

RUN chown -R agones /home/agones && \
    chmod o+x /home/agones/controller

USER agones
ENTRYPOINT ["/home/agones/controller"]