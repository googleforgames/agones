FROM debian:stretch

RUN apt-get update && apt-get install -y unzip curl
RUN useradd -m xonotic

WORKDIR /home/xonotic
RUN curl -o xonotic.zip http://dl.xonotic.org/xonotic-0.8.2.zip && \
    unzip xonotic.zip && rm xonotic.zip

#
# To learn about setting up a Xonotic dedicated server, read the `readme.txt` in the
# `server` directory after downloading the archive from xonotic.org
#

RUN mkdir -p ./.xonotic/data && mv /home/xonotic/Xonotic/server/server_linux.sh /home/xonotic/Xonotic/
COPY ./bin/sdk .
COPY run.sh .
COPY server.cfg ./.xonotic/data
RUN chown -R xonotic:xonotic . && chmod +x run.sh && chmod +x sdk

USER xonotic
ENTRYPOINT /home/xonotic/run.sh