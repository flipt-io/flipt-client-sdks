# Flipt Client Python

[![flipt-client-python](https://img.shields.io/pypi/v/flipt-client.svg)](https://pypi.org/project/flipt-client)

The `flipt-client-python` library contains the Python source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
pip install flipt-client
```

## How Does It Work?

The `flipt-client-python` library is a wrapper around the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

All evaluation happens within the SDK, using the shared library built from the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

Because the evaluation happens within the SDK, the SDKs can be used in environments where the Flipt server is not available or reachable after the initial data is fetched.

## Data Fetching

Upon instantiation, the `flipt-client-python` library will fetch the flag state from the Flipt server and store it in memory. This means that the first time you use the SDK, it will make a request to the Flipt server.

### Polling (Default)

By default, the SDK will poll the Flipt server for new flag state at a regular interval. This interval can be configured using the `update_interval` option when constructing a client. The default interval is 120 seconds.

### Streaming (Flipt Cloud/Flipt v2)

[Flipt Cloud](https://flipt.io/cloud) and [Flipt v2](https://docs.flipt.io/v2) users can use the `streaming` fetch method to stream flag state changes from the Flipt server to the SDK.

When in streaming mode, the SDK will connect to the Flipt server and open a persistent connection that will remain open until the client is closed. The SDK will then receive flag state changes in real-time.

### Retries

The SDK will automatically retry fetching (or initiating streaming) flag state if the client is unable to reach the Flipt server temporarily.

The SDK will retry up to 3 times with an exponential backoff interval between retries. The base delay is 1 second and the maximum delay is 30 seconds.

Retriable errors include:

- `429 Too Many Requests`
- `502 Bad Gateway`
- `503 Service Unavailable`
- `504 Gateway Timeout`
- Other potential transient network or DNS errors

## Supported Architectures

This SDK currently supports the following OSes/architectures:

- Linux x86_64
- Linux arm64
- MacOS x86_64
- MacOS arm64
- Windows x86_64

## Migration Notes

### Pre-1.0.0 -> 1.0.0

This section is for users who are migrating from a previous (pre-1.0.0) version of the SDK.

- `FliptEvaluationClient` has been renamed to `FliptClient`.
- The `namespace` argument is now part of `ClientOptions` as `namespace` (and optionally `environment`).
- `update_interval` and `request_timeout` are now `timedelta` objects instead of `int`.
- All response models are now Pydantic models with explicit types and improved docstrings.
- Backwards compatibility with Pydantic v1 is now maintained.
- All errors now inherit from `FliptError` and use specific subclasses like `ValidationError` and `EvaluationError`.
- The client and models now follow Python naming conventions (snake_case for methods and fields).
- The minimum supported Python version is now 3.8.

## Usage

```python
from flipt_client import FliptClient
from flipt_client.models import ClientOptions, ClientTokenAuthentication

client = FliptClient(
    opts=ClientOptions(
        url="http://localhost:8080",
        authentication=ClientTokenAuthentication(client_token="your-token"),
    )
)

variant_result = client.evaluate_variant(flag_key="flag1", entity_id="entity", context={"fizz": "buzz"})
print(variant_result)

# Important: always close the client to release resources
client.close()
```

### Constructor Arguments

The `FliptClient` constructor accepts a single `opts` argument:

- `opts`: An instance of the `ClientOptions` class. The structure is:
  - `environment`: The environment name. Defaults to `default`.
  - `namespace`: The namespace name. Defaults to `default`.
  - `url`: The URL of the upstream Flipt instance. Defaults to `http://localhost:8080`.
  - `request_timeout`: Timeout for requests. Defaults to no timeout.
  - `update_interval`: Interval to fetch new flag state. Defaults to 120s.
  - `authentication`: Authentication strategy. See [Authentication](#authentication).
  - `reference`: The namespace or reference key. Defaults to `default`.
  - `fetch_mode`: Fetch mode (`polling` or `streaming`). Defaults to polling.
  - `error_strategy`: Error strategy (`fail` or `fallback`). Defaults to fail.
  - `snapshot`: A base64 encoded snapshot of the engine state. Defaults to `None`. See [Snapshotting](#snapshotting).

### Authentication

The `FliptClient` supports:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### Error Handling

All errors raised by the client derive from `flipt_client.errors.FliptError`:

- `ValidationError`: Raised for invalid arguments or configuration.
- `EvaluationError`: Raised for evaluation failures (e.g., flag not found).

Example:

```python
from flipt_client.errors import ValidationError, EvaluationError

try:
    client.evaluate_variant(flag_key=None, entity_id="entity")
except ValidationError as e:
    print(f"Validation error: {e}")
except EvaluationError as e:
    print(f"Evaluation error: {e}")
```

### Snapshotting

The client supports snapshotting of flag state as well as seeding the client with a snapshot for evaluation. This is helpful if you want to use the client in an environment where the Flipt server is not guaranteed to be available or reachable on startup.

To get the snapshot for the client, you can use the `get_snapshot` method. This returns a base64 encoded JSON string that represents the flag state for the client.

You can set the snapshot for the client using the `snapshot` option when constructing a client.

**Note:** You most likely will want to also set the `error_strategy` to `fallback` when using snapshots. This will ensure that you wont get an error if the Flipt server is not available or reachable even on the initial fetch.

You also may want to store the snapshot in a local file so that you can use it to seed the client on startup.

> [!IMPORTANT]
> If the Flipt server becomes reachable after the setting the snapshot, the client will replace the snapshot with the new flag state from the Flipt server.

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
