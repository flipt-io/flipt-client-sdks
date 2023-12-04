# Flipt Client Node

The `flipt-client-node` directory contains the TypeScript source code for a Flipt evaluation client.

## Instructions

To use this client, you can run the following command from the root of the repository:

```bash
cargo build --release
```

This should generate a `target/` directory in the root of this repository, which contains the dynamic linking library built for your platform. This dynamic library will contain the functionality necessary for the Node client to make FFI calls. You'll need to set the `FLIPT_ENGINE_LIB_PATH` environment variable depending on your platform::

The `path/to/lib` will be the path to the dynamic library which will have the following paths depending on your platform.

- **Linux**: `{REPO_ROOT}/target/release/libfliptengine.so`
- **Windows**: `{REPO_ROOT}/target/release/libfliptengine.dll`
- **MacOS**: `{REPO_ROOT}/target/release/libfliptengine.dylib`

You can then use the client like so:

```typescript
import { FliptEvaluationClient } from 'flipt-client-node';

// namespace is the first positional argument and is optional here and will have a value of "default" if not specified.
// engine_opts is the second positional argument and is also optional, the structure is:
// {
//  "url": "http://localhost:8080",
//  "update_interval": 120 
// }
//
// You can replace the url with where your upstream Flipt instance points to, and the update interval for how long you are willing
// to wait for updated flag state.
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
