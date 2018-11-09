cmake_minimum_required (VERSION 3.1.3)

project(agonessdk C CXX)

find_package(Git)

include(ExternalProject)

# Enable C++11 support
set(CMAKE_CXX_STANDARD 11)
set(GRPC_TARGET_NAME gRPC_LIBRARY)

# Build gRPC codebase
ExternalProject_Add(${GRPC_TARGET_NAME}
  PREFIX grpc
    GIT_REPOSITORY "https://github.com/grpc/grpc.git"
    GIT_TAG "v1.16.0"
    CMAKE_ARGS
      -DCMAKE_BUILD_TYPE=Release
    CMAKE_CACHE_ARGS
      -DCMAKE_INSTALL_PREFIX:PATH=${CMAKE_CURRENT_BINARY_DIR}/grpc
      -DgRPC_BUILD_TESTS:BOOL=OFF
      -DgRPC_BUILD_CODEGEN:BOOL=OFF
      -DgRPC_BUILD_CSHARP_EXT:BOOL=OFF
      -DgRPC_BACKWARDS_COMPATIBILITY_MODE:BOOL=OFF
      -DgRPC_INSTALL:BOOL=OFF
)

# Have CMake output libraries to ./lib
set(CMAKE_LIBRARY_OUTPUT_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}/lib/${CMAKE_BUILD_TYPE})
set(CMAKE_ARCHIVE_OUTPUT_DIRECTORY ${CMAKE_CURRENT_BINARY_DIR}/lib/${CMAKE_BUILD_TYPE})

# Agones SDK source files
set(AGONES_SDK_SOURCES
  ${CMAKE_CURRENT_SOURCE_DIR}/sdk.cc
  ${CMAKE_CURRENT_SOURCE_DIR}/sdk.pb.cc
  ${CMAKE_CURRENT_SOURCE_DIR}/sdk.grpc.pb.cc
  ${CMAKE_CURRENT_SOURCE_DIR}/google/api/annotations.pb.cc
  ${CMAKE_CURRENT_SOURCE_DIR}/google/api/http.pb.cc
)

# Agones SDK library target
add_library(${PROJECT_NAME} SHARED ${AGONES_SDK_SOURCES})

# Header directories
include_directories(
  ${CMAKE_CURRENT_SOURCE_DIR}
  ${CMAKE_CURRENT_SOURCE_DIR}/grpc/include
)

# Make sure we build gRPC before we build Agones
add_dependencies(${PROJECT_NAME} ${GRPC_TARGET_NAME})

# gRPC dependency libraries
set(GRPC_LIBRARIES
  ${CMAKE_CURRENT_BINARY_DIR}/grpc/lib/libprotobuf.a
  ${CMAKE_CURRENT_BINARY_DIR}/grpc/lib/libcares.a
  ${CMAKE_CURRENT_BINARY_DIR}/grpc/lib/libz.a
  ${CMAKE_CURRENT_BINARY_DIR}/grpc/src/${GRPC_TARGET_NAME}-build/libaddress_sorting.a
  ${CMAKE_CURRENT_BINARY_DIR}/grpc/src/${GRPC_TARGET_NAME}-build/libgpr.a
  ${CMAKE_CURRENT_BINARY_DIR}/grpc/src/${GRPC_TARGET_NAME}-build/libgrpc++_unsecure.a
  ${CMAKE_CURRENT_BINARY_DIR}/grpc/src/${GRPC_TARGET_NAME}-build/libgrpc_unsecure.a
)

# Link against grpc dependencies
target_link_libraries(${PROJECT_NAME} PRIVATE ${GRPC_LIBRARIES})
