# Flipt Engine WASM

![Status: Beta](https://img.shields.io/badge/status-beta-yellow)
[![npm](https://img.shields.io/npm/v/@flipt-io/flipt-engine-wasm?label=%40flipt-io%2Fflipt-engine-wasm)](https://www.npmjs.com/package/@flipt-io/flipt-engine-wasm)

This is the WebAssembly layer for the Flipt client-side SDKs. It is written in Rust and exposes a JavaScript API for the SDKs to use.

It leverages the [flipt-evaluation](../flipt-evaluation) library to perform the actual evaluation.

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md).

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [wasm-pack](https://rustwasm.github.io/wasm-pack/installer/)
- [wasm-bindgen](https://rustwasm.github.io/wasm-bindgen/)

### Build

```bash
wasm-pack build --target nodejs
```

### Test

```bash
cd integration
npm install
npm test
```
