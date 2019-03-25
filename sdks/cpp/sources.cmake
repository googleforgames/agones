set(SOURCE_FILES
  src/agones/sdk.cc
  src/agones/sdk.grpc.pb.cc
  src/agones/sdk.pb.cc
  
  src/google/annotations.pb.cc
  src/google/http.pb.cc
)

set(HEADER_FILES
  include/agones/sdk.h
  include/agones/sdk.grpc.pb.h
  include/agones/sdk.pb.h
)

set(GOOGLE_HEADER_FILES
  include/google/api/annotations.pb.h
  include/google/api/http.pb.h
)

set(ALL_FILES
  ${SOURCE_FILES}
  ${HEADER_FILES}
  ${GOOGLE_HEADER_FILES}
  "${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_NAME}_export.h"
  "${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_NAME}_global.h"
)
