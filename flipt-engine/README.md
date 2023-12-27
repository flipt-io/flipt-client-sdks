# Flipt Client Engine

This is the client engine for Flipt. It is responsible for evaluating context provided by the native language client SDKs and returning the results of the evaluation.

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md).

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [cbindgen](https://github.com/mozilla/cbindgen)

### Build

```bash
cargo build --release
```

There are some language SDKs that might require a C file header which has the definitions of the functions accessible through the FFI layer. The `cargo build` command will use a build script to generate this header file by way of the [cbindgen](https://github.com/mozilla/cbindgen) tool.

### Test

```bash
cargo test
```
