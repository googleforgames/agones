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
# Makefile for testing and building the C++ SDK
#

#  __     __         _       _     _
#  \ \   / /_ _ _ __(_) __ _| |__ | | ___ ___
#   \ \ / / _` | '__| |/ _` | '_ \| |/ _ \ __|
#    \ V / (_| | |  | | (_| | |_) | |  __\__ \
#     \_/ \__,_|_|  |_|\__,_|_.__/|_|\___|___/
#

CXX = g++
CPPFLAGS += -I/usr/local/include -I$(build_path) -pthread -fPIC
CXXFLAGS += -std=c++11
LDFLAGS += -L/usr/local/lib -lgrpc++_unsecure -lgrpc -lprotobuf -lpthread -ldl

# Directory that this Makefile is in.
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
build_path := $(dir $(mkfile_path))

#   _____                    _
#  |_   _|_ _ _ __ __ _  ___| |_ ___
#    | |/ _` | '__/ _` |/ _ \ __/ __|
#    | | (_| | | | (_| |  __/ |_\__ \
#    |_|\__,_|_|  \__, |\___|\__|___/
#                 |___/

# build the library
build: ensure-bin libagonessdk

# install into /usr/local in the appropriate places
# make sure to install protoc and grpc from source first to build dependencies
install:
	cp $(build_path)/bin/libagonessdk.* /usr/local/lib
	-mkdir -p /usr/local/include/agones
	cp $(build_path)/*.h /usr/local/include/agones/
	-mkdir -p /usr/local/include/google/api
	cp -r $(build_path)/google/api/*.h /usr/local/include/google/api/
	ldconfig

# make the .so build
libagonessdk: google/api/annotations.pb.o google/api/http.pb.o sdk.grpc.pb.o sdk.pb.o sdk.o
	$(CXX) $^ $(LDFLAGS) -shared -o $(build_path)/bin/$@.so
	ar rcs $(build_path)/bin/$@.a $^

# make sure the bin directory exists
ensure-bin:
	-mkdir $(build_path)/bin

# build dev and runtime tarballs
archive: VERSION = "dev"
archive:
	-rm $(build_path)/bin/agonessdk-$(VERSION)-src.zip
	-rm $(build_path)/bin/agonessdk-$(VERSION)-dev-linux-arch_64.tar.gz
	-rm $(build_path)/bin/agonessdk-$(VERSION)-runtime-linux-arch_64.tar.gz
	cp /usr/local/lib/libgrpc.so.6 $(build_path)/bin/
	cp /usr/local/lib/libprotobuf.so.15 $(build_path)/bin/
	cp /usr/local/lib/libagonessdk.so $(build_path)/bin/
	cp /usr/local/lib/libgpr.so.6 $(build_path)/bin/
	cp /usr/local/lib/libgrpc_unsecure.so.6 $(build_path)/bin/
	cd $(build_path)/bin && tar cvf agonessdk-$(VERSION)-runtime-linux-arch_64.tar.gz *
	cd /usr/local && tar cvf $(build_path)/bin/agonessdk-$(VERSION)-dev-linux-arch_64.tar.gz lib include
	cd $(build_path) && zip ./bin/agonessdk-$(VERSION)-src.zip Makefile *.md *.cc *.h

clean:
	-rm -r $(build_path)/bin
	-find -name '*.o' -delete
