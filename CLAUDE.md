# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the Flipt Client SDKs repository - a collection of client-side evaluation SDKs for the Flipt feature flag system. The repository contains multiple language implementations that wrap Rust-based evaluation engines.

## Architecture

The project uses a multi-layered architecture:

1. **Core Evaluation Engine** (`flipt-evaluation/`): Rust library containing the core evaluation logic
2. **Engine Implementations**:
   - **FFI Engine** (`flipt-engine-ffi/`): For native platform integration via Foreign Function Interface
   - **WASM Engine** (`flipt-engine-wasm/`): WebAssembly for non-JS environments
   - **WASM JS Engine** (`flipt-engine-wasm-js/`): WebAssembly optimized for JavaScript environments
3. **Language Clients**: Thin wrappers in various languages that interface with the engines

## Supported Languages

- **FFI-based**: Python, Ruby, Java, C#, Dart/Flutter, Kotlin (Android), Swift (iOS)
- **WASM-based**: Go, JavaScript, Node.js, React, Browser

### FFI-based Clients

- **Python** (`flipt-client-python/`): Python SDK using ctypes for FFI bindings
- **Ruby** (`flipt-client-ruby/`): Ruby gem with FFI bindings
- **Java** (`flipt-client-java/`): Java SDK with JNI bindings
- **C#** (`flipt-client-csharp/`): .NET SDK with P/Invoke bindings
- **Dart** (`flipt-client-dart/`): Dart/Flutter SDK with dart:ffi bindings
- **Kotlin Android** (`flipt-client-kotlin-android/`): Android-specific Kotlin SDK
- **Swift** (`flipt-client-swift/`): iOS/macOS SDK with Swift Package Manager support

### WASM-based Clients

- **Go** (`flipt-client-go/`): Go SDK embedding WASM module directly
- **Node.js** (`flipt-client-node/`): Node.js SDK using WASM-JS engine (deprecated: consolidated into `flipt-client-js`)
- **Browser** (`flipt-client-browser/`): Browser JavaScript SDK (deprecated: consolidated into `flipt-client-js`)
- **React** (`flipt-client-react/`): React hooks and components
- **JavaScript** (`flipt-client-js/`): Core JavaScript utilities and types shared by browser/node

## Common Development Commands

### Rust (Core Engines)

```bash
# Build all Rust components
cargo build --release

# Run tests
cargo test --all-features

# Format code
cargo fmt --all

# Lint code
cargo clippy --all -- -D warnings

# Test WASM JS engine specifically
cd flipt-engine-wasm-js && wasm-pack test --node
```

### Integration Tests

```bash
# Run all integration tests
dagger --progress=plain run go run ./test

# Run specific SDK tests
dagger --progress=plain run go run ./test -sdks python,ruby

# Run by implementation group
dagger --progress=plain run go run ./test -groups ffi,wasm,js
```

Note: Use `progress=plain` to see the output of the tests.

## Test Groups

- `ffi`: Python, Ruby, Java, Dart, C# (FFI-based SDKs)
- `wasm`: Go (Direct WASM)
- `js`: Node, Browser, React (JavaScript WASM bindings)

## File Structure

- `flipt-evaluation/`: Core evaluation logic in Rust
- `flipt-engine-*/`: Engine implementations (FFI, WASM, WASM-JS)
- `flipt-client-*/`: Language-specific client implementations
- `test/`: Integration test infrastructure using Dagger
- `package/`: Build tooling for packaging engines
- `release/`: Release automation scripts

## Key Implementation Patterns

### Client Interface

All clients implement a standard interface:

- `EvaluateVariant`: Evaluates multi-variant flags
- `EvaluateBoolean`: Evaluates boolean flags
- `EvaluateBatch`: Evaluates multiple flags in one call
- `ListFlags`: Returns available flags
- `GetSnapshot`: Returns current flag state (if supported)
- `Close`: Cleanup method

### Engine Integration

- **FFI clients**: Link to native libraries, handle state polling/streaming
- **WASM clients**: Embed WASM module, handle state fetching from client side

### State Management

All SDKs support two modes for keeping flag state up-to-date:

1. **Polling**: Periodically fetch state at configured intervals
   - Default: 120 seconds
   - Configurable via `update_interval` option
   - Simple and reliable

2. **Streaming** (FFI-based only): Real-time updates via Server-Sent Events (SSE)
   - Instant flag updates
   - Automatic reconnection on failure
   - Lower latency but requires persistent connection

## Development Workflow

1. Changes to core evaluation logic go in `flipt-evaluation/`
2. Engine changes go in respective `flipt-engine-*/` directories
3. Client changes go in `flipt-client-*/` directories
4. Use integration tests to verify cross-language compatibility
5. Create a new branch for your changes
6. Follow conventional commits for commit messages prefer shorter commit messages over longer ones
7. Create a PR for your changes using the `gh pr create` command. Give detailed description of the changes and why you made them.

## Build Dependencies

- Rust toolchain (for engines)
- wasm-pack (for WASM builds)
- Docker (for integration tests)
- Dagger CLI v0.17.1+ (for integration tests)
- Language-specific toolchains as needed
- Platform-specific requirements:
  - **Linux**: musl-gcc for static linking
  - **Mobile**: Xcode (iOS), Android SDK

## Release Process

### Versioning

- The engines versions are managed in the `Cargo.toml` files

- Each SDK maintains its own version in language-specific files:
  - Python: `pyproject.toml`
  - JavaScript/TypeScript: `package.json`
  - Java/Kotlin: `build.gradle`
  - Ruby: `flipt_client.gemspec`
  - Go: Git tags only
  - Rust: `Cargo.toml`

### Release Workflow

### Engines

1. Bump the version in the `Cargo.toml` file
2. Create a PR for the changes
3. Merge the PR
4. Create a tag for the release: `flipt-engine-v{version}`
5. GitHub Actions automatically builds and publishes the engine to this repository

**Note**: The engines must be built and published to this repository before they can be used by the client SDKs!

#### Client SDKs

1. Use the interactive release script: `python release/release.py`
2. Select SDK and version bump type
3. Push the generated tag: `flipt-client-{language}-v{version}`
4. GitHub Actions automatically builds and publishes to package registries

## CI/CD Structure

### Workflow Files

- **Testing**: `test-{sdk}-sdk.yml` - Runs on PRs
- **Packaging**: `package-{sdk}-sdk.yml` - Runs on tag push
- **Engine builds**: Automated for Rust components

## Adding a New SDK

To add a new language SDK:

1. **Choose the engine type**:
   - FFI: For languages with native library support (C interop)
   - WASM: For languages that can run WebAssembly

2. **Create the client directory**: `flipt-client-{language}/`

3. **Implement the standard interface**:
   - `EvaluateVariant`, `EvaluateBoolean`, `EvaluateBatch`
   - `ListFlags`, `Close`
   - State management (polling/streaming)

4. **Add integration tests**:
   - Create test file in the language-specific test directory
   - Update `test/main.go` to include new SDK
   - Add to appropriate test group (ffi/wasm/js)

5. **Set up CI/CD**:
   - Create `.github/workflows/test-{language}-sdk.yml`
   - Create `.github/workflows/package-{language}-sdk.yml`
   - Add Dependabot configuration

6. **Update documentation**:
   - Add to this file (CLAUDE.md)
   - Create SDK-specific README
   - Add to main repository README
