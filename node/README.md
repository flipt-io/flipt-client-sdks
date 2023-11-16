# Flipt Client Node
The `flipt-client-node` directory contains the TypeScript source code for a Flipt evaluation client using FFI to make calls to a core built in Rust.

## Instructions

To use this client, you can run the following command from the root of the repository:

```bash
cargo build
```

This should generate a `target/` directory in the root of this repository, which contains the dynamic linking library built for your platform. This dynamic library will contain the functionality necessary for the Node client to make FFI calls. You'll need to set the `FLIPT_ENGINE_LIB_PATH` environment variable depending on your platform::

The `path/to/lib` will be the path to the dynamic library which will have the following paths depending on your platform.

- **Linux**: `{REPO_ROOT}/target/debug/libengine.so`
- **Windows**: `{REPO_ROOT}/target/debug/libengine.dll`
- **MacOS**: `{REPO_ROOT}/target/debug/libengine.dylib`

You can then use the client like so:

```typescript
import { FliptEvaluationClient } from 'flipt-client-node';

// namespace is optional here and will have a value of "default" if not specified.
const fliptEvaluationClient = new FliptEvaluationClient("staging");

const variant = fliptEvaluationClient.variant("flag1", "someentity", {"fizz": "buzz"});

console.log(variant);
```

## Memmory Management

Since TypeScript/JavaScript is a garbage collected language there is no concept of "freeing" memory. We had to allocated memory for the engine through the `initialize_engine` FFI call, and with that being said please make sure to call the `freeEngine` method on the `FliptEvaluationClient` class once you are done using it.
