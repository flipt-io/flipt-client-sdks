# Flipt Client Python

The `flipt-client-python` directory contains the Python source code for a Flipt evaluation client using FFI to make calls to a core built in Rust.

## Instructions

To use this client, you can run the following command from the root of the repository:

```bash
cargo build --release
```

This should generate a `target/` directory in the root of this repository, which contains the dynamic linking library built for your platform. This dynamic library will contain the functionality necessary for the Python client to make FFI calls. You'll need to set the `FLIPT_ENGINE_LIB_PATH` environment variable depending on your platform:

- **Linux**: `{REPO_ROOT}/target/release/libfliptengine.so`
- **Windows**: `{REPO_ROOT}/target/release/libfliptengine.dll`
- **MacOS**: `{REPO_ROOT}/target/release/libfliptengine.dylib`

In your Python code you can import this client and use it as so:

```python
from flipt_client_python import FliptEvaluationClient

# namespace is optional here and will have a value of "default" if not specified
flipt_evaluation_client = FliptEvaluationClient(namespace="staging")

variant_result = flipt_evaluation_client.variant(flag_key="flag1", entity_id="entity", context={"fizz": "buzz"})

print(variant_result)
```
