#!/bin/sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

cp "$SCRIPT_DIR/include/flipt_engine.h" "$SCRIPT_DIR/flipt_engine.h"
gcc -fPIC -c "$SCRIPT_DIR/wrapper.c" -o "$SCRIPT_DIR/wrapper.o"
gcc -shared -o "$SCRIPT_DIR/libfliptengine.so" "$SCRIPT_DIR/wrapper.o" -L"$SCRIPT_DIR" -lfliptengine -ldl -lpthread