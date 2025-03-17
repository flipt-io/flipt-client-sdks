# Flipt Client C\#

[![flipt-client-c#](https://img.shields.io/nuget/v/flipt.client)](https://www.nuget.org/packages/Flipt.Client/)

The `flipt-client-csharp` library contains the C# source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

```bash
dotnet add package Flipt.Client
```

## How Does It Work?

The `flipt-client-csharp` library is a wrapper around the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

All evaluation happens within the SDK, using the shared library built from the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

Because the evaluation happens within the SDK, the SDKs can be used in environments where the Flipt server is not available or reachable after the initial data is fetched.

## Data Fetching

Upon instantiation, the `flipt-client-csharp` library will fetch the flag state from the Flipt server and store it in memory. This means that the first time you use the SDK, it will make a request to the Flipt server.

### Polling (Default)

By default, the SDK will poll the Flipt server for new flag state at a regular interval. This interval can be configured using the `FetchMode` option when constructing a client. The default interval is 120 seconds.

### Streaming (Flipt Cloud Only)

[Flipt Cloud](https://flipt.io/cloud) users can use the `streaming` fetch method to stream flag state changes from the Flipt server to the SDK.

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

## Usage

In your C# code you can import this client and use it as so:

```csharp
using Flipt.Client;

var options = new ClientOptions
  {
    Url = "http://localhost:8080",
    Authentication = new Authentication { ClientToken = "secret" }
  };

var client = new EvaluationClient("default", options);

var context = new Dictionary<string, string> { { "fizz", "buzz" } };
var response = client.EvaluateVariant("flag1", "someentity", context);
```

### Initialization Arguments

The `EvaluationClient` constructor accepts two optional arguments:

- `Namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `Options`: An instance of the `ClientOptions` type that supports several options for the client. The structure is:
  - `Url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
  - `Authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
  - `Reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
  - `FetchMode`: The fetch mode to use when fetching flag state. If not provided, the client will default to polling.
  - `ErrorStrategy`: The error strategy to use when fetching flag state. If not provided, the client will default to `Fail`. See the [Error Strategies](#error-strategies) section for more information.

### Authentication

The `EvaluationClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### Error Strategies

The `EvaluationClient` supports the following error strategies:

- `Fail`: The client will throw an error if the flag state cannot be fetched. This is the default behavior.
- `Fallback`: The client will maintain the last known good state and use that state for evaluation in case of an error.

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
