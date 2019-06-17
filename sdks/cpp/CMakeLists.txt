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
    if(MessageType STREQUAL FATAL_ERROR OR
      MessageType STREQUAL SEND_ERROR OR
      MessageType STREQUAL WARNING OR
      MessageType STREQUAL AUTHOR_WARNING)
      list(REMOVE_AT ARGV 0)
      _message(${MessageType} "${ARGV}")
    endif()
  endfunction()  
  
  set(CMAKE_INSTALL_MESSAGE NEVER)
  set(CMAKE_VERBOSE_MAKEFILE OFF)
  set_property(GLOBAL PROPERTY RULE_MESSAGES OFF)
  set_property(GLOBAL PROPERTY TARGET_MESSAGES OFF)
endif(AGONES_SILENT_OUTPUT)

# Project AGONES
project(agones VERSION 0.9.0 HOMEPAGE_URL https://github.com/googleforgames/agones LANGUAGES C CXX)

# Build options
option(AGONES_BUILD_SHARED "Build Agones C++ SDK as dynamic library" OFF)
option(AGONES_FORCE_GRPC_VERSION "Build Agones C++ SDK only with officially supported gRPC version" ON)
option(AGONES_CREATE_PACKAGE "Generate CMake installation step and package files for Agones C++ SDK" ON)

# Currently we doesn't support build time generation of proto/grpc files,
# so gRPC version should be strict
set(AGONES_FORCE_GRPC_VERSION ON)

# Settings
list(APPEND CMAKE_MODULE_PATH "${CMAKE_CURRENT_LIST_DIR}/cmake")

set_property(GLOBAL PROPERTY USE_FOLDERS ON)

set(CMAKE_CXX_STANDARD 14)
set(CMAKE_CXX_STANDARD_REQUIRED TRUE)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_INCLUDE_CURRENT_DIR ON)
set(CMAKE_INCLUDE_CURRENT_DIR_IN_INTERFACE ON)
set(CMAKE_DEBUG_POSTFIX "d")
set(AGONES_GRPC_VERSION "1.16.1")

if (AGONES_BUILD_SHARED)
  add_compile_definitions(
    PROTOBUF_USE_DLLS
    LIBPROTOBUF_EXPORTS
  )
endif()

# --> Connecting gRPC
macro(validate_path IN_PATH OUT_RESULT)
  set(${OUT_RESULT} FALSE)
  if (NOT IS_DIRECTORY ${${IN_PATH}})
	file(TO_CMAKE_PATH "$ENV{${IN_PATH}}" ${IN_PATH})
  else()
    file(TO_CMAKE_PATH "${${IN_PATH}}" ${IN_PATH})
  endif()
  if (IS_DIRECTORY ${${IN_PATH}})
    set(${OUT_RESULT} TRUE)
  endif()
endmacro(validate_path)

set(GRPC_IS_INSTALLED FALSE)
set(GRPC_PATH_FOUND FALSE)
validate_path(grpc_SOURCE_DIR GRPC_PATH_FOUND)
if (GRPC_PATH_FOUND)
  # Checking if we use gRPC package or gRPC source folder
  # Usually it should be findXXX.cmake or xxxConfig.cmake files, but for gRPC we need to check gRPCTargets.cmake too
  if (EXISTS ${grpc_SOURCE_DIR}/gRPCTargets.cmake)
    set(GRPC_IS_INSTALLED TRUE)
	find_package(gRPC CONFIG REQUIRED)
  endif()
else()
  # Download gRPC, if necessary
  include(Fetch_gRPC)
endif()

add_subdirectory(${grpc_SOURCE_DIR} ${CMAKE_CURRENT_BINARY_DIR}/grpc EXCLUDE_FROM_ALL)

# Check gRPC version
get_directory_property(GRPC_VERSION DIRECTORY ${grpc_SOURCE_DIR} DEFINITION PACKAGE_VERSION)
if (NOT GRPC_VERSION STREQUAL AGONES_GRPC_VERSION)
  set(GRPC_VERSION_SEVERITY WARNING)
  if (AGONES_FORCE_GRPC_VERSION)
    set(GRPC_VERSION_SEVERITY FATAL_ERROR)
  endif()
  message(${GRPC_VERSION_SEVERITY} "Agones SDK currently supports only \"${AGONES_GRPC_VERSION}\" version. Building with \"${GRPC_VERSION}\" may cause unexpected problems.")
endif()

# Configuring
set(AGONES_DEPENDENCIES libprotobuf grpc++_unsecure)
set(AGONES_GRPC_TARGETS address_sorting c-ares gpr grpc_unsecure grpc++_unsecure libprotobuf zlibstatic)
foreach(THIRDPARTY_TARGET ${AGONES_GRPC_TARGETS})
  set_property(TARGET ${THIRDPARTY_TARGET} PROPERTY FOLDER third_party)
endforeach()
# <-- Connecting gRPC

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

if (MSVS)
  set(OPT_DISABLE_COMPILER_WARNINGS /wd4101 /wd4146 /wd4251 /wd4661)
  target_compile_options(libprotobuf PUBLIC ${OPT_DISABLE_COMPILER_WARNINGS})
  target_compile_options(${PROJECT_NAME} PUBLIC ${OPT_DISABLE_COMPILER_WARNINGS})
else()
  set(OPT_DISABLE_COMPILER_WARNINGS -wd4101 -wd4146 -wd4251 -wd4661)
endif()

# Agones SDK
include(./sources.cmake)

set(AGONES_BUILD_TYPE STATIC)
if (AGONES_BUILD_SHARED)
  set(AGONES_BUILD_TYPE SHARED)
endif()

add_library(${PROJECT_NAME} ${AGONES_BUILD_TYPE} ${ALL_FILES})
target_link_libraries(${PROJECT_NAME} PRIVATE ${AGONES_DEPENDENCIES})

target_include_directories(${PROJECT_NAME} PUBLIC
  $<BUILD_INTERFACE:${CMAKE_CURRENT_LIST_DIR}/include>
  $<BUILD_INTERFACE:${CMAKE_CURRENT_LIST_DIR}/include/${PROJECT_NAME}>
  $<INSTALL_INTERFACE:include>
)

# Fix compiler warnings
# https://github.com/protocolbuffers/protobuf/blob/master/cmake/README.md#notes-on-compiler-warnings

# Export header
include(GenerateExportHeader)
set(EXPORT_HEADER "${PROJECT_NAME}_export.h")
generate_export_header(${PROJECT_NAME} EXPORT_FILE_NAME ${EXPORT_HEADER} DEFINE_NO_DEPRECATED)

# Global header
set(GLOBAL_HEADER "${PROJECT_NAME}_global.h")
set(GLOBAL_CONFIG_CONTENT "")
if (AGONES_BUILD_SHARED)
  set(GLOBAL_CONFIG_CONTENT "\
// Dynamic linkage require macro for protobuf\n\
#ifndef PROTOBUF_USE_DLLS\n\
#define PROTOBUF_USE_DLLS\n\
#endif\n"
  )
endif()
configure_file(cmake/${GLOBAL_HEADER}.in ${GLOBAL_HEADER} @ONLY)

if(MSVC)
  add_definitions(/FI"${GLOBAL_HEADER}")
else()
  # GCC or Clang
  add_definitions(-include ${GLOBAL_HEADER})
endif()

# CMake package generation
include(CMakePackageConfigHelpers)

if (AGONES_CREATE_PACKAGE)
  set(_INCLUDE_DIRS "include")
  set(_CMAKE_CONFIG_DESTINATION "cmake")

  # If gRPC is built from source (not from package), then we need to redistribute gRPC and it dependencies
  if (NOT GRPC_IS_INSTALLED)
    # gRPC headers
    install(DIRECTORY ${grpc_SOURCE_DIR}/include/ DESTINATION ${_INCLUDE_DIRS})
	# Protobuf headers
	install(DIRECTORY ${grpc_SOURCE_DIR}/third_party/protobuf/src/ DESTINATION ${_INCLUDE_DIRS} FILES_MATCHING PATTERN "*.h")
	# gRPC and it dependencies
	if (NOT AGONES_BUILD_SHARED)
      install(TARGETS ${AGONES_GRPC_TARGETS} EXPORT gRPC
        LIBRARY DESTINATION  lib
        ARCHIVE DESTINATION  lib
        RUNTIME DESTINATION  bin
        INCLUDES DESTINATION ${_INCLUDE_DIRS}
      )
	endif()
  endif()
  
  # Config for find_package
  configure_package_config_file(
    cmake/${PROJECT_NAME}Config.cmake.in
    ${PROJECT_NAME}Config.cmake
    INSTALL_DESTINATION ${CMAKE_INSTALL_PREFIX}/${_CMAKE_CONFIG_DESTINATION}
    PATH_VARS _INCLUDE_DIRS PROJECT_VERSION
	NO_SET_AND_CHECK_MACRO
  )
  # Build artifacts
  install(TARGETS ${PROJECT_NAME} ${AGONES_GRPC_TARGETS} EXPORT ${PROJECT_NAME}
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
    FILES ${HEADER_FILES} "${CMAKE_CURRENT_BINARY_DIR}/${EXPORT_HEADER}" "${CMAKE_CURRENT_BINARY_DIR}/${GLOBAL_HEADER}"
    DESTINATION include/${PROJECT_NAME}
  )
  # Google header files
  install(
    FILES ${GOOGLE_HEADER_FILES}
    DESTINATION include/google/api
  )
  # PDB file
  if (AGONES_BUILD_SHARED)
    install(FILES $<TARGET_PDB_FILE:${PROJECT_NAME}> DESTINATION bin CONFIGURATIONS Debug OPTIONAL)
    install(FILES $<TARGET_PDB_FILE:${AGONES_GRPC_TARGETS}> DESTINATION bin CONFIGURATIONS Debug OPTIONAL)
  endif()

  unset(_INCLUDE_DIRS)
  unset(_CMAKE_CONFIG_DESTINATION)
else() # Package is created with absolute pathes in build folder
  # Use agones and thirdparty include directories
  set(_INCLUDE_DIRS
    "${CMAKE_CURRENT_LIST_DIR}/include"
    "${grpc_SOURCE_DIR}/include"
    "${grpc_SOURCE_DIR}/third_party/protobuf/src"
  )
  set(_CMAKE_CONFIG_DESTINATION "${CMAKE_BINARY_DIR}")

  configure_package_config_file(
    cmake/${PROJECT_NAME}Config.cmake.in
    "${CMAKE_BINARY_DIR}/${PROJECT_NAME}Config.cmake"
    INSTALL_DESTINATION ${_CMAKE_CONFIG_DESTINATION}
    PATH_VARS _INCLUDE_DIRS PROJECT_VERSION
    NO_SET_AND_CHECK_MACRO
  )
  export(TARGETS ${PROJECT_NAME} ${AGONES_GRPC_TARGETS} FILE "${CMAKE_BINARY_DIR}/${PROJECT_NAME}Targets.cmake")
  # Register SDK local artifacts in system https://cmake.org/cmake/help/latest/manual/cmake-packages.7.html#package-registry-example
  #export(PACKAGE ${PROJECT_NAME})

  unset(_INCLUDE_DIRS)
  unset(_CMAKE_CONFIG_DESTINATION)
endif(AGONES_CREATE_PACKAGE)
