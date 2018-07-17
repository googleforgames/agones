# Copyright 2017 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#
# Makefile for building the world's simplest C++ game server
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

CXX = g++
CPPFLAGS += -I/usr/local/include -pthread
CXXFLAGS += -std=c++11
LDFLAGS += -L/usr/local/lib -lagonessdk -lgrpc++_unsecure -lgrpc -lprotobuf -lpthread -ldl
REPOSITORY = gcr.io/agones-images

# Directory that this Makefile is in.
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
project_path := $(dir $(mkfile_path))
server_tag = $(REPOSITORY)/cpp-simple-server:0.3

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# build the library
build: ensure-bin server-dynamic server-static

# Build a docker image for the server, and tag it
build-image:
	docker build $(project_path) --tag=$(server_tag)

# make the server executable
server-dynamic: server.o
	$(CXX) $^ $(LDFLAGS) -o $(project_path)bin/$@

# make the server executable
server-static: server.o
	$(CXX) $^ $(LDFLAGS) -o $(project_path)bin/$@ -static

# make sure the bin directory exists
ensure-bin:
	-mkdir $(project_path)/bin

clean:
	-rm -r $(project_path)/bin
	-rm *.o
