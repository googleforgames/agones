# Start from this base image, based on Ubuntu LTS.
FROM unitymultiplay/linux-base-image:1.0.1
ENV SERVER_BUILD_NAME=Server
ENV SERVER_BUILD_PATH=Builds/${SERVER_BUILD_NAME}

# Install dependencies.
USER root
RUN apt-get update && apt-get install -y
USER mpukgame

# Set up a working directory.
WORKDIR /game

# Add your game files and perform any required init steps.
COPY ${SERVER_BUILD_PATH}/${SERVER_BUILD_NAME}_Data/ ./${SERVER_BUILD_NAME}_Data/
COPY ${SERVER_BUILD_PATH}/${SERVER_BUILD_NAME}.x86_64 .
COPY ${SERVER_BUILD_PATH}/UnityPlayer.so .

# Set your game entrypoint.
CMD sleep 2 && ./Server.x86_64
