# Flipt Client Python

The `flipt-client-python` directory contains the Python source code for the Flipt client-side evaluation client.

## Installation

Currently, to use this client, you'll need to build the dynamic library and the package locally and install it. This is a temporary solution until we can figure out a better way to package and distribute the libraries.

The dynamic library will contain the functionality necessary for the client to make calls to the Flipt engine via FFI. See [flipt-engine](../flipt-engine) for more information on the Flipt engine and FFI.

### Prerequisites

- [Rust](https://www.rust-lang.org/tools/install)
- [Python](https://www.python.org/downloads/)
- [Make](https://www.gnu.org/software/make/)
- [poetry](https://python-poetry.org/docs/#installation)

### Automated Build

1. Build and copy the dynamic library to the `flipt-client-python/ext` directory for your platform. This will also build and install the `flipt-client` Python package. You can do this by running the following command from the root of the repository:

    ```bash
    make python
    ```

### Manual Build

1. Build the Rust dynamic library

    ```bash
    cargo build --release

This should generate a `target/` directory in the root of this repository, which contains the dynamically linked library built for your platform.

2. You'll need to copy the dynamic library to the `flipt-client-python/ext/$OS_$ARCH/` directory. This is a temporary solution until we can figure out a better way to package the libraries with the package.

The `path/to/lib` will be the path to the dynamic library which will have the following paths depending on your platform.

- **Linux**: `{REPO_ROOT}/target/release/libfliptengine.so`
- **Windows**: `{REPO_ROOT}/target/release/libfliptengine.dll`
- **MacOS**: `{REPO_ROOT}/target/release/libfliptengine.dylib`

3. You can then build the package and install it locally. You can do this by running the following command from the `flipt-client-python` directory:

    ```bash
    poetry install
    ```

## Usage

In your Python code you can import this client and use it as so:

```python
from flipt_client import FliptEvaluationClient

# "namespace" and "engine_opts" are two keyword arguments that this constructor accepts
# namespace: which namespace to fetch flag state from
# engine_opts: follows the model EngineOpts in the models.py file. Configures the url of the upstream Flipt instance, the interval in which to fetch new flag state, and the auth token if your upstream Flipt instance requires it
flipt_evaluation_client = FliptEvaluationClient()

variant_result = flipt_evaluation_client.evaluate_variant(flag_key="flag1", entity_id="entity", context={"fizz": "buzz"})

print(variant_result)
```
