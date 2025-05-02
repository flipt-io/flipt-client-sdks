#!/bin/bash
set -euxo pipefail

# Check if target argument is provided
if [ $# -ne 1 ]; then
  echo "Error: Target argument is required"
  echo "Usage: $0 [linux-x86_64|linux-aarch64]"
  exit 1
fi

# Convert TARGET to lowercase using tr
TARGET=$(echo "$1" | tr '[:upper:]' '[:lower:]')
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$SCRIPT_DIR/.."
RUST_FLAGS="RUSTFLAGS=-Zlocation-detail=none -Zfmt-debug=none"

case $TARGET in
"linux-x86_64")
  if ! command -v musl-gcc &> /dev/null; then
      echo "Error: musl-gcc is not installed. Please install it first."
      exit 1
  fi
  rustup default nightly
  rustup target add x86_64-unknown-linux-musl

  # https://github.com/johnthagen/min-sized-rust
  $RUST_FLAGS cargo +nightly build -p flipt-engine-ffi --release --target=x86_64-unknown-linux-musl

  mkdir -p "/tmp/ffi"

  mv "$ROOT_DIR/target/x86_64-unknown-linux-musl/release/libfliptengine.a" "/tmp/ffi/libfliptengine_static.a"

  musl-gcc -shared -o "$ROOT_DIR/target/x86_64-unknown-linux-musl/release/libfliptengine.so" -fPIC "$SCRIPT_DIR/wrapper.c" \
      -I"$SCRIPT_DIR/include" \
      -L"/tmp/ffi" -lfliptengine_static \
      -Wl,-Bstatic -static-libgcc -static

  rm -rf "/tmp/ffi"
  ;;

"linux-aarch64")
  if ! command -v musl-gcc &> /dev/null; then
      echo "Error: musl-gcc is not installed. Please install it first."
      exit 1
  fi
  rustup default nightly
  rustup target add aarch64-unknown-linux-musl

  # https://github.com/johnthagen/min-sized-rust
  $RUST_FLAGS cargo +nightly build -p flipt-engine-ffi --release --target=aarch64-unknown-linux-musl

  mkdir -p "/tmp/ffi"

  mv "$ROOT_DIR/target/aarch64-unknown-linux-musl/release/libfliptengine.a" "/tmp/ffi/libfliptengine_static.a"

  musl-gcc -shared -o "$ROOT_DIR/target/aarch64-unknown-linux-musl/release/libfliptengine.so" -fPIC "$SCRIPT_DIR/wrapper.c" \
      -I"$SCRIPT_DIR/include" \
      -L"/tmp/ffi" -lfliptengine_static \
      -Wl,-Bstatic -static-libgcc -static

  rm -rf "/tmp/ffi"
  ;;
*)
  echo "Error: Invalid Target specified"
  echo "Usage: $0 [linux-x86_64|linux-aarch64]"
  exit 1
  ;;
esac