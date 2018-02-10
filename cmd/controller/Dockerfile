FROM alpine:3.7

COPY ./bin/controller /home/agones/controller
COPY ./certs /home/agones/certs
RUN apk --update add ca-certificates && \
                adduser -D agones && \
                chown -R agones /home/agones && \
                chmod o+x /home/agones/controller

USER agones
ENTRYPOINT ["/home/agones/controller"]