# Integration Tests

The different languages clients should all have an integration test suite that is dependent on the dynamic library being present somewhere and a running instance of Flipt. When running the integration tests with Dagger, the dynamic library will be built in a separate container and copied into the target machine for your tests automatically.

In the `test/` directory we will use [Dagger](https://dagger.io/) to orchestrate setting up the dependencies for running the test suites for the different languages.

## Requirements

Make sure you have `dagger` installed. This module is pinned to `v0.14.0` currently in CI.

> [!IMPORTANT]
> We recommend installing the same version of Dagger as is used in CI. Follow the [Dagger Installation Instructions](https://docs.dagger.io/install/#stable-release) to install the correct version (v0.14.0).

## Running Tests

From the root of this repository you can run:

```bash
dagger run go run ./test
```

This will run integration tests for every language that is supported. You can filter which tests to run in two ways:

1. By specific languages using the `-sdks` flag which accepts a comma-separated list of values:

```bash
dagger run go run ./test -sdks python,ruby
```

2. By groups using the `-groups` flag which accepts a comma-separated list of values:

```bash
dagger run go run ./test -groups ffi,js
```

The available groups are:

- `ffi`: Python, Ruby, Java, Dart, C# (SDKs using FFI bindings)
- `wasm`: Go (SDKs using WebAssembly directly)
- `js`: Node, Browser, React (SDKs using JavaScript WASM bindings)

You can also combine both flags to run specific SDKs and groups together.
