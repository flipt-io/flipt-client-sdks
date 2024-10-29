#!/bin/bash

cargo build -p flipt-engine-ffi --release --target=aarch64-apple-ios-sim
cargo build -p flipt-engine-ffi --release --target=aarch64-apple-ios
cargo build -p flipt-engine-ffi --release --target=aarch64-apple-darwin

rm -rf ../flipt-client-swift/Sources/FliptEngineFFI.xcframework

xcodebuild -create-xcframework \
        -library ../target/aarch64-apple-ios-sim/release/libfliptengine.a -headers ./include/ \
        -library ../target/aarch64-apple-ios/release/libfliptengine.a -headers ./include/ \
        -library ../target/aarch64-apple-darwin/release/libfliptengine.a -headers ./include/ \
        -output "../flipt-client-swift/Sources/FliptEngineFFI.xcframework"
