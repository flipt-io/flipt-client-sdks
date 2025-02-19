#!/bin/bash
set -euxo pipefail

if ! command -v musl-gcc &> /dev/null; then
    echo "Error: musl-gcc is not installed. Please install it first."
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

cd "$SCRIPT_DIR"
# Rename the static library
mv "libfliptengine.a" "libfliptengine_static.a"

musl-gcc -shared -o libfliptengine.so -fPIC wrapper.c \
    -I./include \
    -L. -lfliptengine_static \
    -Wl,-Bstatic -static-libgcc -static

rm libfliptengine_static.a