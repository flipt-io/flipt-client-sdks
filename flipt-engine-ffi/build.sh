#!/bin/bash
set -euxo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

cd "$SCRIPT_DIR"
# Copy and rename the static library
cp "libfliptengine.a" "libfliptengine_static.a"

musl-gcc -shared -o libfliptengine.so -fPIC wrapper.c \
    -L. -lfliptengine_static \
    -Wl,-Bstatic -static-libgcc -static
