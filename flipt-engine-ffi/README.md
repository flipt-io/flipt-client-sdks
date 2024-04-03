# Flipt Engine FFI

This is the FFI layer for the Flipt client-side SDKs. It is written in Rust and exposes a C API for the SDKs to use.

It leverages the [flipt-evaluation](../flipt-evaluation) library to perform the actual evaluation.

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md).

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [cbindgen](https://github.com/mozilla/cbindgen)

### Build

```bash
cargo build --release
```

There are some language SDKs that might require a C file header which has the definitions of the functions accessible through the FFI layer.

### Test

```bash
cargo test
```
