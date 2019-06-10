# Copyright 2018 Google LLC All Rights Reserved.
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

cmake_minimum_required (VERSION 3.13.0)

# Silent cmake/build output (internal option)
# Extra command line options
# For windows: cmake --build . --config Release --target install -- /v:quiet /nologo
# For *nix: cmake --build . --target install -- -s
option(AGONES_SILENT_OUTPUT "Show only warnings/error messages" OFF)
if (AGONES_SILENT_OUTPUT)
    function(message)
        list(GET ARGV 0 MessageType)
        list(REMOVE_AT ARGV 0)
        if (MessageType STREQUAL FATAL_ERROR OR
            MessageType STREQUAL SEND_ERROR OR
            MessageType STREQUAL WARNING OR
            MessageType STREQUAL AUTHOR_WARNING OR
            NOT ${AGONES_SILENT_OUTPUT}
            )
            _message(${MessageType} "${ARGV}")
        endif()
    endfunction()

    set(CMAKE_INSTALL_MESSAGE NEVER)
    set(CMAKE_VERBOSE_MAKEFILE OFF)
    set_property(GLOBAL PROPERTY RULE_MESSAGES OFF)
    set_property(GLOBAL PROPERTY TARGET_MESSAGES OFF)
endif(AGONES_SILENT_OUTPUT)

# Project AGONES
project(agones VERSION 0.11.0 HOMEPAGE_URL https://github.com/GoogleCloudPlatform/agones LANGUAGES C CXX)

# Build options
option(AGONES_FORCE_GRPC_VERSION "Build Agones C++ SDK only with supported gRPC version" ON)

# Currently we doesn't support build time generation of proto/gRPC files,
# so gRPC version should be strict
set(AGONES_FORCE_GRPC_VERSION ON)
set(AGONES_GRPC_VERSION_MATCH "")
if (AGONES_FORCE_GRPC_VERSION)
    set(AGONES_GRPC_VERSION_MATCH EXACT)
endif()

# Settings
set_property(GLOBAL PROPERTY USE_FOLDERS ON)

set(CMAKE_CXX_STANDARD 14)
set(CMAKE_CXX_STANDARD_REQUIRED TRUE)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_INCLUDE_CURRENT_DIR ON)
set(CMAKE_INCLUDE_CURRENT_DIR_IN_INTERFACE ON)
set(CMAKE_DEBUG_POSTFIX "d")
set(AGONES_GRPC_VERSION "1.16.1")

# Third party
include(./cmake/prerequisites.cmake)
find_package(ZLIB REQUIRED)
find_package(Protobuf REQUIRED CONFIG)
find_package(gRPC ${AGONES_GRPC_VERSION} ${AGONES_GRPC_VERSION_MATCH} REQUIRED CONFIG)

# Platform specific stuff
if (WIN32)
    # Windows
    add_compile_definitions(
        _WIN32_WINNT=0x0600
        WINDOWS
    )
elseif (APPLE)
    # Mac OS
elseif (UNIX AND NOT APPLE)
    # Linux
endif()

# Agones SDK
include(./sources.cmake)

# Global header
set(GLOBAL_HEADER "${PROJECT_NAME}_global.h")
set(GLOBAL_CONFIG_CONTENT "\n")
configure_file(cmake/${GLOBAL_HEADER}.in ${GLOBAL_HEADER} @ONLY)

if(MSVC)
  add_definitions(/FI"${GLOBAL_HEADER}")
else()
  # GCC or Clang
  add_definitions(-include ${GLOBAL_HEADER})
endif()

set(AGONES_DEPENDENCIES protobuf::libprotobuf gRPC::grpc++_unsecure)

add_library(${PROJECT_NAME} STATIC ${ALL_FILES})
target_link_libraries(${PROJECT_NAME} PUBLIC ${AGONES_DEPENDENCIES})

target_include_directories(${PROJECT_NAME} PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_LIST_DIR}/include>
    $<BUILD_INTERFACE:${CMAKE_CURRENT_LIST_DIR}/include/${PROJECT_NAME}>
    $<INSTALL_INTERFACE:include>
)

# Fix compiler warnings
# https://github.com/protocolbuffers/protobuf/blob/master/cmake/README.md#notes-on-compiler-warnings
if (MSVC)
    set(OPT_DISABLE_COMPILER_WARNINGS /wd4101 /wd4146 /wd4251 /wd4661)
    target_compile_options(${PROJECT_NAME} PUBLIC ${OPT_DISABLE_COMPILER_WARNINGS})
endif()

# Export header
include(GenerateExportHeader)
set(EXPORT_HEADER "${PROJECT_NAME}_export.h")
generate_export_header(${PROJECT_NAME} EXPORT_FILE_NAME ${EXPORT_HEADER} DEFINE_NO_DEPRECATED)

# CMake package generation
include(CMakePackageConfigHelpers)

set(_INCLUDE_DIRS "include")
set(_CMAKE_CONFIG_DESTINATION "cmake")

# Config for find_package
configure_package_config_file(
    cmake/${PROJECT_NAME}Config.cmake.in
    ${PROJECT_NAME}Config.cmake
    INSTALL_DESTINATION ${CMAKE_INSTALL_PREFIX}/${_CMAKE_CONFIG_DESTINATION}
    PATH_VARS _INCLUDE_DIRS PROJECT_VERSION
    NO_SET_AND_CHECK_MACRO
)

# Build artifacts
install(TARGETS ${PROJECT_NAME} EXPORT ${PROJECT_NAME}
    LIBRARY DESTINATION  lib
    ARCHIVE DESTINATION  lib
    RUNTIME DESTINATION  bin
    INCLUDES DESTINATION ${_INCLUDE_DIRS}
)
install(EXPORT ${PROJECT_NAME} DESTINATION ${_CMAKE_CONFIG_DESTINATION} FILE ${PROJECT_NAME}Targets.cmake)
# Package config
install(
    FILES ${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_NAME}Config.cmake
    DESTINATION ${_CMAKE_CONFIG_DESTINATION}
)
# Agones header files
install(
    FILES ${HEADER_FILES} "${CMAKE_CURRENT_BINARY_DIR}/${EXPORT_HEADER}"
    DESTINATION include/${PROJECT_NAME}
)
# Google header files
install(
    FILES ${GOOGLE_HEADER_FILES}
    DESTINATION include/google/api
)

unset(_INCLUDE_DIRS)
unset(_CMAKE_CONFIG_DESTINATION)
