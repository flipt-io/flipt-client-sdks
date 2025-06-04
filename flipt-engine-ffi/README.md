# Flipt Engine FFI

![GitHub Release](https://img.shields.io/github/v/release/flipt-io/flipt-client-sdks?filter=flipt-engine-ffi-*)

This is the FFI layer for the Flipt client-side SDKs. It is written in Rust and exposes a C API for the SDKs to use.

It leverages the [flipt-evaluation](../flipt-evaluation) library to perform the actual evaluation.

## Architecture

See [ARCHITECTURE.md](../ARCHITECTURE.md).

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)

### Build

```bash
cargo build -p flipt-engine-ffi --release
```

There are some language SDKs that might require a C file header which has the definitions of the functions accessible through the FFI layer.

We use [cbindgen](https://github.com/mozilla/cbindgen) to generate the C file header automatically at build time. The generated header is located in the `include` directory.

### Logging

The FFI layer uses the `log` and `env_logger` crates for logging. The `FLIPT_ENGINE_LOG` environment variable can be used to set the log level.

```bash
export FLIPT_ENGINE_LOG=debug
```

### Test

```bash
cargo test -p flipt-engine-ffi
```
