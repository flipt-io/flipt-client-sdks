# Flipt Engine WASM

This is the WebAssembly layer for the Flipt client-side SDKs. It is written in Rust and exposes a JavaScript API for the SDKs to use.

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md).

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [wasm-pack](https://rustwasm.github.io/wasm-pack/installer/)
- [wasm-bindgen](https://rustwasm.github.io/wasm-bindgen/)

### Build

```bash
wasm-pack build --target web
```
