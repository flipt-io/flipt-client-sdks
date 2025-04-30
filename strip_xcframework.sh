#!/bin/bash

# Exit on error
set -eou pipefail

# Path to the original XCFramework
XCFRAMEWORK_PATH="staging/ios/FliptEngineFFI.xcframework"

# Path for the stripped version
STRIPPED_PATH="staging/ios/FliptEngineFFI_stripped.xcframework"

# Create output directory structure
mkdir -p "$STRIPPED_PATH/ios-arm64/Headers"
mkdir -p "$STRIPPED_PATH/ios-arm64-simulator/Headers"

# Copy Info.plist
cp "$XCFRAMEWORK_PATH/Info.plist" "$STRIPPED_PATH/"

# Copy headers
cp "$XCFRAMEWORK_PATH/ios-arm64/Headers/"* "$STRIPPED_PATH/ios-arm64/Headers/"
cp "$XCFRAMEWORK_PATH/ios-arm64-simulator/Headers/"* "$STRIPPED_PATH/ios-arm64-simulator/Headers/"

# Strip the libraries
echo "Stripping device library..."
strip -x "$XCFRAMEWORK_PATH/ios-arm64/libfliptengine.a" -o "$STRIPPED_PATH/ios-arm64/libfliptengine.a"

echo "Stripping simulator library..."
strip -x "$XCFRAMEWORK_PATH/ios-arm64-simulator/libfliptengine.a" -o "$STRIPPED_PATH/ios-arm64-simulator/libfliptengine.a"

# Print file size comparison
echo "Original sizes:"
ls -lh "$XCFRAMEWORK_PATH/ios-arm64/libfliptengine.a" "$XCFRAMEWORK_PATH/ios-arm64-simulator/libfliptengine.a"

echo "Stripped sizes:"
ls -lh "$STRIPPED_PATH/ios-arm64/libfliptengine.a" "$STRIPPED_PATH/ios-arm64-simulator/libfliptengine.a"

mv $STRIPPED_PATH $XCFRAMEWORK_PATH

echo "Done! Stripped XCFramework is at $XCFRAMEWORK_PATH"
