#!/bin/bash
set -euxo pipefail

# Check if target argument is provided
if [ $# -ne 1 ]; then
  echo "Error: Target argument is required"
  echo "Usage: $0 [swift|android|linux-x86_64|linux-aarch64]"
  exit 1
fi

TARGET=$1
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$SCRIPT_DIR/.."

case $TARGET in
"swift")
  rustup target add aarch64-apple-ios x86_64-apple-ios aarch64-apple-ios-sim

  cargo build -p flipt-engine-ffi --release --target=aarch64-apple-ios-sim
  cargo build -p flipt-engine-ffi --release --target=aarch64-apple-ios
  cargo build -p flipt-engine-ffi --release --target=aarch64-apple-darwin

  rm -rf "$ROOT_DIR/flipt-client-swift/Sources/FliptEngineFFI.xcframework"

  xcodebuild -create-xcframework \
    -library "$ROOT_DIR/target/aarch64-apple-ios-sim/release/libfliptengine.a" -headers "$SCRIPT_DIR/include/" \
    -library "$ROOT_DIR/target/aarch64-apple-ios/release/libfliptengine.a" -headers "$SCRIPT_DIR/include/" \
    -library "$ROOT_DIR/target/aarch64-apple-darwin/release/libfliptengine.a" -headers "$SCRIPT_DIR/include/" \
    -output "$ROOT_DIR/flipt-client-swift/Sources/FliptEngineFFI.xcframework"
  ;;

"android")
  rustup target add x86_64-linux-android aarch64-linux-android

  cargo build -p flipt-engine-ffi --release --target=x86_64-linux-android
  cargo build -p flipt-engine-ffi --release --target=aarch64-linux-android

  mkdir -p "$ROOT_DIR/flipt-client-kotlin-android/src/main/cpp/libs/x86_64"
  mkdir -p "$ROOT_DIR/flipt-client-kotlin-android/src/main/cpp/libs/arm64-v8a"
  cp "$ROOT_DIR/target/x86_64-linux-android/release/libfliptengine.a" "$ROOT_DIR/flipt-client-kotlin-android/src/main/cpp/libs/x86_64/libfliptengine.a"
  cp "$ROOT_DIR/target/aarch64-linux-android/release/deps/libfliptengine.so" "$ROOT_DIR/flipt-client-kotlin-android/src/main/cpp/libs/arm64-v8a/libfliptengine.a"
  cp -r "$SCRIPT_DIR/include/" "$ROOT_DIR/flipt-client-kotlin-android/src/main/cpp/include"
  ;;

"linux-x86_64")
  if ! command -v musl-gcc &> /dev/null; then
      echo "Error: musl-gcc is not installed. Please install it first."
      exit 1
  fi
  rustup target add x86_64-unknown-linux-musl

  cargo build -p flipt-engine-ffi --release --target=x86_64-unknown-linux-musl

  mkdir -p /tmp/ffi
  mkdir -p "$ROOT_DIR/output"
  cp "$ROOT_DIR/target/x86_64-unknown-linux-musl/release/libfliptengine.a" "$ROOT_DIR/output/libfliptengine_static.a"

  musl-gcc -shared -o "/tmp/ffi/libfliptengine.so" -fPIC "$SCRIPT_DIR/wrapper.c" \
      -I"$SCRIPT_DIR/include" \
      -L"$ROOT_DIR/output" -lfliptengine_static \
      -Wl,-Bstatic -static-libgcc -static

  rm -rf "$ROOT_DIR/output"
  ;;

"linux-aarch64")
  if ! command -v musl-gcc &> /dev/null; then
      echo "Error: musl-gcc is not installed. Please install it first."
      exit 1
  fi

  rustup target add aarch64-unknown-linux-musl

  cargo build -p flipt-engine-ffi --release --target=aarch64-unknown-linux-musl

  mkdir -p /tmp/ffi
  mkdir -p "$ROOT_DIR/output"
  cp "$ROOT_DIR/target/aarch64-unknown-linux-musl/release/libfliptengine.a" "$ROOT_DIR/output/libfliptengine_static.a"

  musl-gcc -shared -o "/tmp/ffi/libfliptengine.so" -fPIC "$SCRIPT_DIR/wrapper.c" \
      -I"$SCRIPT_DIR/include" \
      -L"$ROOT_DIR/output" -lfliptengine_static \
      -Wl,-Bstatic -static-libgcc -static

  rm -rf "$ROOT_DIR/output"
  ;;
*)
  echo "Error: Invalid Target specified"
  echo "Usage: $0 [swift|android|linux-x86_64|linux-aarch64]"
  exit 1
  ;;
esac