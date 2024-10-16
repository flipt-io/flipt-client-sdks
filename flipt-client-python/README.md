# Flipt Client Python

[![pypi](https://img.shields.io/pypi/v/flipt-client.svg)](https://pypi.org/project/flipt-client)

The `flipt-client-python` library contains the Python source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
pip install flipt-client
```

## Supported Architectures

This SDK currently supports the following OSes/architectures:

- Linux x86_64
- Linux arm64
- MacOS x86_64
- MacOS arm64
- Windows x86_64

## Usage

In your Python code you can import this client and use it as so:

```python
from flipt_client import FliptEvaluationClient

# "namespace" and "engine_opts" are two keyword arguments that this constructor accepts
# namespace: which namespace to fetch flag state from
# engine_opts: follows the model EngineOpts in the models.py file. Configures the url of the upstream Flipt instance, the interval in which to fetch new flag state, and the authentication method if your upstream Flipt instance requires it
flipt_evaluation_client = FliptEvaluationClient()

variant_result = flipt_evaluation_client.evaluate_variant(flag_key="flag1", entity_id="entity", context={"fizz": "buzz"})

print(variant_result)
```

### Constructor Arguments

The `FliptEvaluationClient` constructor accepts two optional arguments:

- `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `engine_opts`: An instance of the `EngineOpts` class that supports several options for the client. The structure is:
  - `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
  - `update_interval`: The interval (in seconds) in which to fetch new flag state. If not provided, the client will default to 120 seconds.
  - `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
  - `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.

### Authentication

The `FliptEvaluationClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
