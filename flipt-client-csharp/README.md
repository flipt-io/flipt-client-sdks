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

- The main client class is now `FliptClient` (was `EvaluationClient`).
- The `Namespace` parameter is now part of `ClientOptions`.
- The `Environment` parameter is now part of `ClientOptions` to support Flipt v2.
- Error handling uses a new hierarchy: `FliptException`, `ValidationException`, `EvaluationException`.

## Usage

```csharp
using FliptClient;
using FliptClient.Models;

var options = new ClientOptions
{
    Url = "http://localhost:8080",
    Namespace = "default",
    Authentication = new Authentication { ClientToken = "your-token" }
};

using var client = new FliptClient(options);

try
{
    var context = new Dictionary<string, string> { { "userId", "123" } };
    var result = client.EvaluateBoolean("my-flag", "entity-id", context);
    Console.WriteLine($"Enabled: {result.Enabled}");
}
catch (ValidationException ex)
{
    // Handle validation errors
}
catch (EvaluationException ex)
{
    // Handle evaluation errors
}
```

### Initialization Arguments

The `FliptClient` constructor accepts one optional argument:

- `Options`: An instance of the `ClientOptions` type that supports several options for the client. The structure is:
  - `Environment`: The environment (Flipt v2) to fetch flag state from. If not provided, the client will default to the `default` environment.
  - `Namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
  - `Url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
  - `RequestTimeout`: The timeout (in seconds) for total request time to the upstream Flipt instance. If not provided, the client will default to no timeout. Note: this only affects polling mode. Streaming mode will have no timeout set.
  - `UpdateInterval`: The interval (in seconds) in which to fetch new flag state. If not provided, the client will default to 120 seconds.
  - `Authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
  - `Reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
  - `FetchMode`: The fetch mode to use when fetching flag state. If not provided, the client will default to polling.
  - `ErrorStrategy`: The error strategy to use when fetching flag state. If not provided, the client will default to `Fail`. See the [Error Strategies](#error-strategies) section for more information.
- `Snapshot`: The initial snapshot to use when instantiating the client. See the [Snapshotting](#snapshotting) section for more information.

### Authentication

The `FliptClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### Error Strategies

The `FliptClient` supports the following error strategies:

- `Fail`: The client will throw an error if the flag state cannot be fetched. This is the default behavior.
- `Fallback`: The client will maintain the last known good state and use that state for evaluation in case of an error.

### Snapshotting

The client supports snapshotting of flag state as well as seeding the client with a snapshot for evaluation. This is helpful if you want to use the client in an environment where the Flipt server is not guaranteed to be available or reachable on startup.

To get the snapshot for the client, you can use the `GetSnapshot` method. This returns a base64 encoded JSON string that represents the flag state for the client.

You can set the snapshot for the client using the `Snapshot` property when constructing a client.

**Note:** You most likely will want to also set the `ErrorStrategy` to `Fallback` when using snapshots. This will ensure that you wont get an error if the Flipt server is not available or reachable even on the initial fetch.

You also may want to store the snapshot in a local file so that you can use it to seed the client on startup.

> [!IMPORTANT]
> If the Flipt server becomes reachable after the setting the snapshot, the client will replace the snapshot with the new flag state from the Flipt server.

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
