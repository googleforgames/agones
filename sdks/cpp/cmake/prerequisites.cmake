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

cmake_minimum_required (VERSION 3.5.0)

option(AGONES_BUILD_THIRDPARTY_DEBUG "Build debug version of thirdparty libraries (MSVC only)" OFF)
set(AGONES_OPENSSL_CONFIG_STRING "VC-WIN64A" CACHE STRING "See https://github.com/openssl/openssl/blob/OpenSSL_1_1_1-stable/INSTALL for details")
set(AGONES_THIRDPARTY_INSTALL_PATH "${CMAKE_INSTALL_PREFIX}" CACHE STRING "Path for installing third-party OpenSSL and gRPC, if they are not found with find_package")

if (NOT DEFINED MSVC)
    set(AGONES_BUILD_THIRDPARTY_DEBUG OFF)
    set(AGONES_OPENSSL_CONFIG_STRING "" CACHE STRING "" FORCE)
endif()

include(ProcessorCount)
ProcessorCount(CPU_COUNT)
if (CPU_COUNT GREATER 0 AND NOT DEFINED CMAKE_BUILD_PARALLEL_LEVEL)
    set($ENV{CMAKE_BUILD_PARALLEL_LEVEL} ${CPU_COUNT})
endif()

# gRPC repo and version
set(gRPC_GIT_REPO "https://github.com/gRPC/gRPC.git")
set(gRPC_GIT_TAG "v1.36.1")

# OpenSSL required only for successful build gRPC
set(OPENSSL_GIT_REPO "https://github.com/openssl/openssl.git")
set(OPENSSL_GIT_TAG "OpenSSL_1_1_1-stable")

include(FetchContent)

function(download_git_repo NAME REPO TAG)
    set(BASE_DIR ${CMAKE_CURRENT_BINARY_DIR}/${NAME})
    set(${NAME}_SOURCE_DIR "${BASE_DIR}/src")
    FetchContent_Declare(
        ${NAME}
        GIT_REPOSITORY      "${REPO}"
        GIT_TAG             "${TAG}"
        PREFIX              ${NAME}
        SOURCE_DIR          "${BASE_DIR}/src"
        BINARY_DIR          "${BASE_DIR}/.bin"
        INSTALL_DIR         "${BASE_DIR}/.install"
        SUBBUILD_DIR        "${BASE_DIR}/.build"
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
            SOURCE_DIR          "${BASE_DIR}/src"
            BINARY_DIR          "${BASE_DIR}/.bin"
            INSTALL_DIR         "${BASE_DIR}/.install"
            SUBBUILD_DIR        "${BASE_DIR}/.build"
            CONFIGURE_COMMAND   ""
            BUILD_COMMAND       ""
            INSTALL_COMMAND     ""
            TEST_COMMAND        ""
        )
    endif()
    set(${NAME}_SOURCE_DIR "${${NAME}_SOURCE_DIR}" CACHE PATH "Source directory for ${NAME}" FORCE)
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
    endif()
    message(${OUTPUT_TYPE} ${output})
endfunction()

function(invoke_cmake_build NAME CMAKELISTS_PATH)
    message(STATUS "Building ${NAME}...")

    # Build directory
    set(BUILD_DIR ${CMAKE_CURRENT_BINARY_DIR}/${NAME}/.bin)
    set(INSTALL_DIR ${AGONES_THIRDPARTY_INSTALL_PATH}/${NAME})
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

    execute_and_check(${BUILD_DIR} ${CMAKE_COMMAND} ${CMAKELISTS_PATH} -G ${CMAKE_GENERATOR} -Wno-dev ${ARG_BUILD_TYPE} -DCMAKE_INSTALL_PREFIX=${INSTALL_DIR} -DCMAKE_PREFIX_PATH=${AGONES_THIRDPARTY_INSTALL_PATH} ${ARGN})

    # Building
    if (${AGONES_BUILD_THIRDPARTY_DEBUG})
        execute_and_check(${BUILD_DIR} ${CMAKE_COMMAND} --build . ${ARG_CONFIG_DEBUG} --target install)
    endif()

    execute_and_check(${BUILD_DIR} ${CMAKE_COMMAND} --build . ${ARG_CONFIG_RELEASE} --target install)
    set(${NAME}_DIR "${INSTALL_DIR}" CACHE PATH "CMake package directory for ${NAME}" FORCE)
endfunction(invoke_cmake_build)

find_package(gRPC CONFIG QUIET)
find_package(OpenSSL QUIET)

# OpenSSL // Required only for gRPC build. Do not build, if gRPC is found.
if (NOT ${OpenSSL_FOUND} AND NOT ${gRPC_FOUND})
    set(OPENSSL_ROOT_DIR "${AGONES_THIRDPARTY_INSTALL_PATH}/OpenSSL" CACHE PATH "OpenSSL root directory" FORCE)
    find_package(OpenSSL QUIET)
    if (NOT ${OpenSSL_FOUND})
        if (${WIN32} AND ${MINGW})
            message(FATAL_ERROR "MinGW build for OpenSSL is not supported, please set OPENSSL_ROOT_DIR variable to OpenSSL path")
        endif()
        download_git_repo(openssl ${OPENSSL_GIT_REPO} ${OPENSSL_GIT_TAG})
        message(STATUS "Building OpenSSL... ${AGONES_OPENSSL_CONFIG_STRING}")
        if (WIN32)
            execute_and_check(${openssl_SOURCE_DIR} perl Configure ${AGONES_OPENSSL_CONFIG_STRING} "--prefix=${OPENSSL_ROOT_DIR}" "--openssldir=${OPENSSL_ROOT_DIR}")
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
set(AGONES_OWN_GRPC FALSE CACHE BOOL "Third party is built by Agones")
if (NOT ${gRPC_FOUND})
    download_git_repo(gRPC ${gRPC_GIT_REPO} ${gRPC_GIT_TAG})
    file(MAKE_DIRECTORY ${AGONES_THIRDPARTY_INSTALL_PATH})
    set(CMAKE_PREFIX_PATH ${CMAKE_PREFIX_PATH} ${AGONES_THIRDPARTY_INSTALL_PATH})

    # Build gRPC prerequisites
    invoke_cmake_build(zlib ${gRPC_SOURCE_DIR}/third_party/zlib)
    set(ZLIB_ROOT "${zlib_DIR}" CACHE PATH "ZLIB_ROOT" FORCE)
    set(ZLIB_PARAM)
    if (AGONES_ZLIB_STATIC)
        find_library(ZLIB_PATH zlibstatic NAMES libzlibstatic PATHS ${ZLIB_ROOT}/lib)
        set(ZLIB_LIBRARY ${ZLIB_PATH} CACHE PATH "ZLIB path" FORCE)
        if (ZLIB_PATH-NOTFOUND)
            message(FATAL_ERROR "Could not locate static zlib path")
        endif()
        set(ZLIB_PARAM "-DZLIB_LIBRARY=${ZLIB_LIBRARY}")
    endif()

    invoke_cmake_build(c-ares ${gRPC_SOURCE_DIR}/third_party/cares/cares
        "-DCARES_STATIC=ON"
        "-DCARES_SHARED=OFF"
    )
    invoke_cmake_build(Protobuf ${gRPC_SOURCE_DIR}/third_party/protobuf/cmake
        "-DZLIB_ROOT=${zlib_DIR}"
        "${ZLIB_PARAM}"
        "-Dprotobuf_MSVC_STATIC_RUNTIME=OFF"
        "-Dprotobuf_BUILD_TESTS=OFF"
    )

    # Build gRPC as cmake package
    set(OPENSSL_PARAM "")
    if (DEFINED OPENSSL_ROOT_DIR)
        set(OPENSSL_PARAM "-DOPENSSL_ROOT_DIR=${OPENSSL_ROOT_DIR}")
    endif()

    set(GRPC_EXTRA_FLAGS)
    if (MINGW)
        set(GRPC_WINVER 0x0600)
        set(GRPC_EXTRA_FLAGS
            "-DCMAKE_C_FLAGS=-D_WIN32_WINNT=${GRPC_WINVER}"
            "-DCMAKE_CXX_FLAGS=-D_WIN32_WINNT=${GRPC_WINVER}"
        )
    endif()

    invoke_cmake_build(gRPC ${gRPC_SOURCE_DIR}
        "${OPENSSL_PARAM}"
        "-DZLIB_ROOT=${zlib_DIR}"
        "${ZLIB_PARAM}"
        "-DgRPC_INSTALL=ON"
        "-DgRPC_BUILD_TESTS=OFF"
        "-DgRPC_BUILD_CSHARP_EXT=OFF"
        "-DgRPC_PROTOBUF_PROVIDER=package"
        "-DgRPC_PROTOBUF_PACKAGE_TYPE=CONFIG"
        "-DgRPC_ZLIB_PROVIDER=package"
        "-DgRPC_CARES_PROVIDER=package"
        "-DgRPC_SSL_PROVIDER=package"
        ${GRPC_EXTRA_FLAGS}
    )
    set(AGONES_OWN_GRPC TRUE CACHE BOOL "Third party is built by Agones" FORCE)
endif()
mark_as_advanced(AGONES_OWN_GRPC)
