# AGENTS.md

This file provides guidance to AI Agents when working with code in this repository.

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

### Linting / Formatting

Before committing, run `mise run lint` from the relevant SDK directory to check formatting and lint:

```bash
cd flipt-client-python && mise run lint
cd flipt-client-go && mise run lint
# etc.
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

- `.mise.toml`: Root mise config (shared tools + monorepo config roots)
- `flipt-evaluation/`: Core evaluation logic in Rust
- `flipt-engine-*/`: Engine implementations (FFI, WASM, WASM-JS)
- `flipt-client-*/`: Language-specific client implementations (each has its own `.mise.toml`)
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

### State Update Callbacks (FFI)

The FFI engine can notify a host after it applies changed flag state, via `set_state_update_callback(engine, callback, user_data)` (`set_state_update_callback_ffi` in the static musl build; see `wrapper.c`). The full contract is in the doc comment in `flipt-engine-ffi/src/lib.rs` and the generated `include/flipt_engine.h`. Java is the reference binding: `addStateUpdateListener` / `onStateUpdate` in `flipt-client-java/src/main/java/io/flipt/client/FliptClient.java`, tests in `StateUpdateListenerTest.java`.

Engine guarantees to rely on, not reimplement:

- Fires only when evaluable state actually changes (`Evaluator::replace_snapshot` returns `changed`). Initial state and transitions into an error state never fire; recovery from an error does.
- Runs on one engine-owned thread, spawned lazily on first registration. Bursts are coalesced into one call.
- Once `set_state_update_callback(NULL)` or `destroy_engine` returns, no callback is running and none will start. `destroy_engine` may be called from inside the callback.
- Changing the callback from inside a callback returns an error (it would deadlock otherwise).

When adding this to another language client, watch for:

1. **Keep the native trampoline alive.** Most FFI layers (JNA, ctypes, Ruby FFI, P/Invoke delegates, Dart `NativeCallable`) free the function pointer when the host object is collected. Hold it strongly until after `destroy_engine` returns, including for clients that are never closed (Java uses a static set). A collected trampoline is a hard crash, not an exception.
2. **Never unwind into the engine.** Catch every exception in the trampoline and log it. A panic or exception crossing the `extern "C"` boundary aborts the process.
3. **Foreign threads.** The callback arrives on a native thread the host runtime didn't create. Check how the runtime handles that: Python needs the GIL (ctypes takes it), Dart must use `NativeCallable.listener` (not `isolateLocal`), Ruby needs `FFI::Function` with a callback thread, C# must keep the delegate referenced. Make the thread daemon/background so it never keeps the process alive.
4. **Register once, fan out in the host.** Register one engine callback lazily and dispatch to a host-side listener list. This keeps add/remove (even from inside a listener) off the engine, and avoids the "change callback from inside callback" error.
5. **Close ordering.** Mark the client closed, call `destroy_engine` (it waits for any in-flight callback), then release the trampoline reference. Don't hold a host lock across `destroy_engine` that a listener might need, or close-from-a-listener deadlocks.
6. **Don't replay initial state.** Listeners only see changes applied after they were added; document that callers should read flags once on registration.

## Development Workflow

1. Changes to core evaluation logic go in `flipt-evaluation/`
2. Engine changes go in respective `flipt-engine-*/` directories
3. Client changes go in `flipt-client-*/` directories
4. Use integration tests to verify cross-language compatibility
5. Create a new branch for your changes
6. Follow conventional commits for commit messages prefer shorter commit messages over longer ones
7. Create a PR for your changes using the `gh pr create` command. Give detailed description of the changes and why you made them.

## Tooling with mise

This monorepo uses [mise](https://mise.jdx.dev/) to manage all development tool versions. Install mise first, then tools are installed automatically when you `cd` into the repo or any SDK subdirectory.

### Setup

```bash
# Install mise (see https://mise.jdx.dev/getting-started.html)
curl https://mise.run | sh

# Trust and install all tools for the repo
mise trust
mise install
```

### How It Works

- **Root `.mise.toml`**: Defines shared tools (Go, Rust, Dagger, wasm-pack, cargo-llvm-cov) and declares SDK subdirectories as monorepo config roots
- **Per-SDK `.mise.toml`**: Each `flipt-client-*/` directory has its own config specifying the language toolchain and a `lint` task

When you `cd` into an SDK directory, mise automatically activates the correct tool versions for that SDK.

### Tools Managed by mise

| Scope                      | Tools                                                           |
| -------------------------- | --------------------------------------------------------------- |
| Root                       | Go 1.25, Rust stable, Dagger 0.18.19, wasm-pack, cargo-llvm-cov |
| Python SDK                 | Python 3.11, Poetry 1.7.0, Black                                |
| Ruby SDK                   | Ruby 3.2                                                        |
| Java SDK                   | Java 21                                                         |
| Kotlin SDK                 | Java 21, ktlint                                                 |
| Go SDK                     | Go 1.25, golangci-lint 2.12.2                                    |
| JS/Node/Browser/React SDKs | Node 22                                                         |
| C# SDK                     | .NET 8                                                          |
| Dart SDK                   | Flutter stable                                                  |
| Swift SDK                  | Swift, swiftformat, swiftlint                                   |

### Lint Tasks

Each SDK defines a `mise run lint` task. Run it from the SDK directory:

```bash
cd flipt-client-python && mise run lint
cd flipt-client-go && mise run lint
# etc.
```

### Build Dependencies (beyond mise)

- Docker (for integration tests)
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

Use the `/release` skill (`.agents/skills/release/SKILL.md`) to audit unreleased changes and determine what needs releasing. It walks through the full process: engine audit, SDK audit, version bumps, tagging, and publishing.

#### Engines

Each engine has its own tag that triggers CI builds:

1. Bump the version in the relevant `Cargo.toml` file(s)
2. Run `cargo check` to verify compilation
3. Create a PR for the changes, merge it
4. Create and push tags from main:
   - `flipt-engine-ffi-v{version}` — triggers CI to build and publish FFI binaries
   - `flipt-engine-wasm-v{version}` — bookkeeping (WASM is bundled with SDKs at release time)
   - `flipt-engine-wasm-js-v{version}` — bookkeeping (WASM-JS is bundled with SDKs at release time)

**Note**: Only the FFI engine tag triggers builds that publish platform binaries. FFI-based SDKs must wait for these builds to complete before releasing. WASM/WASM-JS SDKs can be released immediately.

#### Client SDKs

Two options:

1. **Non-interactive** (preferred for automation): `cd release && .venv/bin/python release_cli.py --sdk flipt-client-go --bump patch --publish`
2. **Interactive**: `cd release && .venv/bin/python release.py`

Both handle version bumps, file updates, tagging, and pushing. See `release/README.md` for full usage.

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

5. **Add mise config**:
   - Create `.mise.toml` in the SDK directory with required tool versions and a `lint` task
   - Add the SDK directory to `config_roots` in the root `.mise.toml`

6. **Set up CI/CD**:
   - Create `.github/workflows/test-{language}-sdk.yml`
   - Create `.github/workflows/package-{language}-sdk.yml`
   - Add Dependabot configuration

7. **Update documentation**:
   - Add to this file (CLAUDE.md)
   - Create SDK-specific README
   - Add to main repository README
