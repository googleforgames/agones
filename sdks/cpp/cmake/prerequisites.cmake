# Copyright 2019 Google LLC All Rights Reserved.
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

option(BUILD_THIRDPARTY_DEBUG "Build debug version of thirdparty libraries (MSVC only)" TRUE)
set(OPENSSL_CONFIG_STRING "VC-WIN64A" CACHE STRING "See https://github.com/openssl/openssl/blob/master/INSTALL for details")

if (NOT MSVC)
    set(BUILD_THIRDPARTY_DEBUG FALSE)
    set(OPENSSL_CONFIG_STRING "" CACHE STRING "" FORCE)
endif()

set(THIRDPARTY_INSTALL_PATH ${CMAKE_INSTALL_PREFIX})

set(gRPC_GIT_REPO "https://github.com/gRPC/gRPC.git")
set(gRPC_GIT_TAG "v1.16.1")

# OpenSSL required only for successful build gRPC
set(OPENSSL_GIT_REPO "https://github.com/openssl/openssl.git")
set(OPENSSL_GIT_TAG "OpenSSL_1_1_1")

include(FetchContent)

function(download_git_repo NAME REPO TAG)
    set(${NAME}_SOURCE_DIR "${CMAKE_CURRENT_BINARY_DIR}/${NAME}/src")
    FetchContent_Declare(
        ${NAME}
        GIT_REPOSITORY      "${REPO}"
        GIT_TAG             "${TAG}"
        PREFIX              ${NAME}
        SOURCE_DIR          "${${NAME}_DIR}"
        BINARY_DIR          "${NAME}/.bin"
        INSTALL_DIR         "${NAME}/.install"
        SUBBUILD_DIR        "${NAME}/.build"
        CONFIGURE_COMMAND   ""
        BUILD_COMMAND       ""
        INSTALL_COMMAND     ""
        TEST_COMMAND        ""
    )
    FetchContent_GetProperties(${NAME})
    if (NOT ${NAME}_POPULATED)
        message("Fetching ${NAME} ${TAG}")
            FetchContent_Populate(
            ${NAME}
            QUIET
            GIT_REPOSITORY      "${REPO}"
            GIT_TAG             "${TAG}"
            PREFIX              ${NAME}
            SOURCE_DIR          "${${NAME}_SOURCE_DIR}"
            BINARY_DIR          "${NAME}/.bin"
            INSTALL_DIR         "${NAME}/.install"
            SUBBUILD_DIR        "${NAME}/.build"
            CONFIGURE_COMMAND   ""
            BUILD_COMMAND       ""
            INSTALL_COMMAND     ""
            TEST_COMMAND        ""
        )
    endif()
    set(${NAME}_SOURCE_DIR "${${NAME}_SOURCE_DIR}" PARENT_SCOPE)
endfunction(download_git_repo)

function(execute_and_check WORKING_DIR)
    execute_process(
        COMMAND ${ARGN}
        WORKING_DIRECTORY ${WORKING_DIR}
        RESULT_VARIABLE result
        OUTPUT_VARIABLE output
        ERROR_VARIABLE output
    )
    set(OUTPUT_TYPE STATUS)
    if (NOT ${result} EQUAL 0)
        set(OUTPUT_TYPE FATAL_ERROR)
        message(SEND_ERROR "Command:${ARGN}\n${result}\n")
    endif()
    message(${OUTPUT_TYPE} ${output})
endfunction()

function(invoke_cmake_build NAME CMAKELISTS_PATH)
    message(STATUS "Building ${NAME}...")

    # Build directory
    set(BUILD_DIR ${CMAKE_CURRENT_BINARY_DIR}/third_party/${NAME})
    set(INSTALL_DIR ${THIRDPARTY_INSTALL_PATH})
    file(MAKE_DIRECTORY ${BUILD_DIR})
    
    # Makefile generation
    set(ARG_BUILD_TYPE "")
    set(ARG_CONFIG_DEBUG "--config" "Debug")
    set(ARG_CONFIG_RELEASE "--config" "Release")
    if (NOT ${CMAKE_BUILD_TYPE} STREQUAL "")
        set(ARG_BUILD_TYPE "-DCMAKE_BUILD_TYPE=${CMAKE_BUILD_TYPE}")
        set(ARG_CONFIG_DEBUG "")
        set(ARG_CONFIG_RELEASE "")
    endif()

    execute_and_check(${BUILD_DIR} ${CMAKE_COMMAND} ${CMAKELISTS_PATH} -G ${CMAKE_GENERATOR} -Wno-dev ${ARG_BUILD_TYPE} -DCMAKE_INSTALL_PREFIX=${INSTALL_DIR} ${ARGN})

    # Building
    if (BUILD_THIRDPARTY_DEBUG)
        execute_and_check(${BUILD_DIR} ${CMAKE_COMMAND} --build . ${ARG_CONFIG_DEBUG} --target install)
    endif()
    
    execute_and_check(${BUILD_DIR} ${CMAKE_COMMAND} --build . ${ARG_CONFIG_RELEASE} --target install)
    set(${NAME}_DIR "${INSTALL_DIR}" PARENT_SCOPE)
endfunction(invoke_cmake_build)

# OpenSSL
find_package(OpenSSL QUIET)

if (NOT ${OpenSSL_FOUND})
    set(OPENSSL_ROOT_DIR "${THIRDPARTY_INSTALL_PATH}/OpenSSL")
    find_package(OpenSSL QUIET)
    if (NOT ${OpenSSL_FOUND})
        download_git_repo(openssl ${OPENSSL_GIT_REPO} ${OPENSSL_GIT_TAG})
        message(STATUS "Building OpenSSL... ${OPENSSL_CONFIG_STRING}")
        if (WIN32)
            execute_and_check(${openssl_SOURCE_DIR} perl Configure ${OPENSSL_CONFIG_STRING} "--prefix=${OPENSSL_ROOT_DIR}" "--openssldir=${OPENSSL_ROOT_DIR}")
            execute_and_check(${openssl_SOURCE_DIR} nmake)
            execute_and_check(${openssl_SOURCE_DIR} nmake install)
        else()
            execute_and_check(${openssl_SOURCE_DIR} "./config" "--prefix=${OPENSSL_ROOT_DIR}" "--openssldir=${OPENSSL_ROOT_DIR}")
            execute_and_check(${openssl_SOURCE_DIR} make)
            execute_and_check(${openssl_SOURCE_DIR} make install)
        endif()
    endif()
endif()

# gRPC
find_package(gRPC QUIET)

if (NOT ${gRPC_FOUND})
    download_git_repo(gRPC ${gRPC_GIT_REPO} ${gRPC_GIT_TAG})

    invoke_cmake_build(zlib ${gRPC_SOURCE_DIR}/third_party/zlib)
    invoke_cmake_build(cares ${gRPC_SOURCE_DIR}/third_party/cares/cares)
    invoke_cmake_build(protobuf ${gRPC_SOURCE_DIR}/third_party/protobuf/cmake
        "-DZLIB_ROOT=${zlib_DIR}"
        "-Dprotobuf_MSVC_STATIC_RUNTIME=OFF"
        "-Dprotobuf_BUILD_TESTS=OFF"
    )
    invoke_cmake_build(gRPC ${gRPC_SOURCE_DIR}
        "-DOPENSSL_ROOT_DIR=${OPENSSL_ROOT_DIR}"
        "-DZLIB_ROOT=${zlib_DIR}"
        "-DgRPC_INSTALL=ON"
        "-DgRPC_BUILD_TESTS=OFF"
        "-DgRPC_PROTOBUF_PROVIDER=package"
        "-DgRPC_ZLIB_PROVIDER=package"
        "-DgRPC_CARES_PROVIDER=package"
        "-DgRPC_SSL_PROVIDER=package"
    )
    
    set(CMAKE_PREFIX_PATH ${CMAKE_PREFIX_PATH} ${CMAKE_INSTALL_PREFIX})
endif()