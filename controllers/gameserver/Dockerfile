FROM alpine:3.6

COPY ./bin/gameserver /home/agon/gameserver
RUN apk --update add ca-certificates && \
                adduser -D agon && \
                chown -R agon /home/agon && \
                chmod o+x /home/agon/gameserver

USER agon
ENTRYPOINT /home/agon/gameserver