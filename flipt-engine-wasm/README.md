# Flipt Engine WASM

This is the WASM layer for the Flipt client-side SDKs. It is written in Rust and compiles to the `wasm32-wasip1` target.

It leverages the [flipt-evaluation](../flipt-evaluation) library to perform the actual evaluation.

## Architecture

See [ARCHITECTURE.md](../ARCHITECTURE.md).

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [wasm-opt](https://github.com/WebAssembly/binaryen/releases) (optional, for optimization)

### Build

```bash
rustup target add wasm32-wasip1
cargo build -p flipt-engine-wasm --release --target wasm32-wasip1
```

### Optimize

We use the `wasm-opt` tool to optimize the WASM module and reduce it's size.

```bash
wasm-opt --converge --flatten --rereloop -Oz -Oz --gufa -o [output] [input]
```

Flags:

- `--converge`: Runs multiple rounds of optimization until further optimizations don’t reduce the binary size.
- `--flatten`: Reduces nested control flow by simplifying nested if-else and loops.
- `--rereloop`: Converts irreducible control flow into structured loops.
- `-Oz`: Aggressive size optimization mode.
- `--gufa`: (Global Unrolling and Function Analysis) Performs function-level optimizations and loop unrolling based on global analysis.

### Test

```bash
cargo test -p flipt-engine-wasm
```
