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
