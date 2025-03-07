# Flipt Engine WASM JS

![GitHub Release](https://img.shields.io/github/v/release/flipt-io/flipt-client-sdks?filter=flipt-engine-wasm-*)

This is the WASM layer for the Flipt JavaScript client-side SDKs. It is written in Rust and exposes a JS API for the SDKs to use.

It leverages the [flipt-evaluation](../flipt-evaluation) library to perform the actual evaluation.

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md).

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [wasm-pack](https://rustwasm.github.io/wasm-pack/installer/)
- [wasm-bindgen](https://rustwasm.github.io/wasm-bindgen/)

### Build

```bash
wasm-pack build --target [bundler|nodejs|web]
```

### Test

```bash
wasm-pack test --node
```
