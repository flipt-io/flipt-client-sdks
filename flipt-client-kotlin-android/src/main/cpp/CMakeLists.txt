cmake_minimum_required(VERSION 3.10.2)
project(fliptengine_wrapper VERSION 1.0)

set(CMAKE_BUILD_RPATH "$ORIGIN")


add_library( libfliptengine STATIC IMPORTED )

if (${ANDROID_ABI} STREQUAL "arm64-v8a")
    set_property(TARGET libfliptengine PROPERTY IMPORTED_LOCATION "${CMAKE_SOURCE_DIR}/libs/arm64-v8a/libfliptengine.a")
elseif (${ANDROID_ABI} STREQUAL "x86_64")
    set_property(TARGET libfliptengine PROPERTY IMPORTED_LOCATION "${CMAKE_SOURCE_DIR}/libs/x86_64/libfliptengine.a")
endif()

add_library(fliptengine_wrapper SHARED fliptengine_wrapper.c)
target_link_libraries(fliptengine_wrapper PRIVATE libfliptengine)

set_target_properties(fliptengine_wrapper PROPERTIES
        VERSION $(PROJECT_VERSION)
        SOVERSION 1
)