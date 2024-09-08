#!/bin/bash

cargo build --release
uniffi-bindgen generate src/flipt_engine.udl --language swift --out-dir bindings/swift
uniffi-bindgen generate src/flipt_engine.udl --language kotlin --out-dir bindings/kotlin