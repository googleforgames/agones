#!/bin/bash

# Configure makefile and build Agones C++ SDK
cmake -DCMAKE_BUILD_TYPE=Debug -G "Unix Makefiles" && make
cmake -DCMAKE_BUILD_TYPE=Release -G "Unix Makefiles" && make
