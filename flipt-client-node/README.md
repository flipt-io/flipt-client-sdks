# Flipt Client Node

The `flipt-client-node` directory contains the TypeScript source code for the Flipt client-side evaluation client.

## Installation

Currently, to use this client, you'll need to build the dynamic library and the Node package locally and install it. This is a temporary solution until we can figure out a better way to package and distribute the libraries.

The dynamic library will contain the functionality necessary for the client to make calls to the Flipt engine via FFI. See [flipt-engine](../flipt-engine) for more information on the Flipt engine and FFI.

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [Node](https://nodejs.org/en/download/)
- [Make](https://www.gnu.org/software/make/)

### Automated Build

1. Build and copy the dynamic library to the `flipt-client-node` directory. You can do this by running the following command from the root of the repository:

    ```bash
    make node
    ```

2. Install the package locally. You can do this by running the following command from the `flipt-client-node` directory:

    ```bash
    npm install -g flipt-client-{version}.tgz
    ```

### Manual Build

1. Build the Rust dynamic library

    ```bash
    cargo build --release
    ```

This should generate a `target/` directory in the root of this repository, which contains the dynamically linked library built for your platform.

2. You'll need to copy the dynamic library to the `flipt-client-node/ext` directory. This is a temporary solution until we can figure out a better way to package the libraries with the package.

The `path/to/lib` will be the path to the dynamic library which will have the following paths depending on your platform.

- **Linux**: `{REPO_ROOT}/target/release/libfliptengine.so`
- **Windows**: `{REPO_ROOT}/target/release/libfliptengine.dll`
- **MacOS**: `{REPO_ROOT}/target/release/libfliptengine.dylib`

3. You can then build the package and install it locally. You can do this by running the following command from the `flipt-client-node` directory:

    ```bash
    npm install
    npm run build
    npm pack
    npm install -g flipt-client-{version}.tgz
    ```

## Usage

In your Node code you can import this client and use it as so:

```typescript
import { FliptEvaluationClient } from 'flipt-client';

// namespace is the first positional argument and is optional here and will have a value of "default" if not specified.
// engine_opts is the second positional argument and is also optional, the structure is:
// {
//  "url": "http://localhost:8080",
//  "update_interval": 120,
//  "auth_token": "secret"
// }
//
// You can replace the url with where your upstream Flipt instance points to, the update interval for how long you are willing
// to wait for updated flag state, and the auth token if your Flipt instance requires it.
const fliptEvaluationClient = new FliptEvaluationClient();

const variant = fliptEvaluationClient.evaluateVariant("flag1", "someentity", {"fizz": "buzz"});

console.log(variant);
```

## Memory Management

Since TypeScript/JavaScript is a garbage collected language there is no concept of "freeing" memory. We have to allocate memory for the engine through the `initialize_engine` FFI call.

Make sure to call the `freeEngine` method on the `FliptEvaluationClient` class once you are done using it.

```typescript
fliptEvaluationClient.freeEngine();
```
