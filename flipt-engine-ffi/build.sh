#!/bin/bash

# Check if SDK argument is provided
if [ $# -ne 1 ]; then
  echo "Error: SDK argument is required"
  echo "Usage: $0 [swift|android]"
  exit 1
fi

SDK=$1

case $SDK in
"swift")
  rustup target add aarch64-apple-ios x86_64-apple-ios aarch64-apple-ios-sim

  cargo build -p flipt-engine-ffi --release --target=aarch64-apple-ios-sim
  cargo build -p flipt-engine-ffi --release --target=aarch64-apple-ios
  cargo build -p flipt-engine-ffi --release --target=aarch64-apple-darwin
  rm -rf ../flipt-client-swift/Sources/FliptEngineFFI.xcframework

  xcodebuild -create-xcframework \
    -library ../target/aarch64-apple-ios-sim/release/libfliptengine.a -headers ./include/ \
    -library ../target/aarch64-apple-ios/release/libfliptengine.a -headers ./include/ \
    -library ../target/aarch64-apple-darwin/release/libfliptengine.a -headers ./include/ \
    -output "../flipt-client-swift/Sources/FliptEngineFFI.xcframework"
  ;;

"android")
  rustup target add x86_64-linux-android aarch64-linux-android
  cargo install cargo-ndk
  cargo ndk -t x86_64 -t arm64-v8a -p 30 build --release
  mkdir -p ../flipt-client-kotlin-android/src/main/cpp/libs/x86_64
  mkdir -p ../flipt-client-kotlin-android/src/main/cpp/libs/arm64-v8a
  cp ../target/x86_64-linux-android/release/libfliptengine.a ../flipt-client-kotlin-android/src/main/cpp/libs/x86_64/libfliptengine.a
  cp ../target/aarch64-linux-android/release/libfliptengine.a ../flipt-client-kotlin-android/src/main/cpp/libs/arm64-v8a/libfliptengine.a
  cp -r include/ ../flipt-client-kotlin-android/src/main/cpp/include
  ;;

*)
  echo "Error: Invalid SDK specified"
  echo "Usage: $0 [swift|android]"
  exit 1
  ;;
esac
