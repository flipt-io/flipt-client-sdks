# Flipt Client Swift

[![flipt-client-swift](https://img.shields.io/github/v/tag/flipt-io/flipt-client-swift?filter=v*&label=flipt-client-swift)](https://github.com/flipt-io/flipt-client-swift)

The `flipt-client-swift` library contains the Swift source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

### Swift Package Manager

Add the following to your `Package.swift` file:

```swift
dependencies: [
    .package(url: "https://github.com/flipt-io/flipt-client-swift.git", from: "1.x.x")
]
```

Or add it directly through Xcode:

1. File > Add Package Dependencies
2. Enter package URL: `https://github.com/flipt-io/flipt-client-swift`

## How Does It Work?

The `flipt-client-swift` library is a wrapper around the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

All evaluation happens within the SDK, using the shared library built from the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

Because the evaluation happens within the SDK, the SDKs can be used in environments where the Flipt server is not available or reachable after the initial data is fetched.

## Data Fetching

Upon instantiation, the `flipt-client-swift` library will fetch the flag state from the Flipt server and store it in memory. This means that the first time you use the SDK, it will make a request to the Flipt server.

### Polling (Default)

By default, the SDK will poll the Flipt server for new flag state at a regular interval. This interval can be configured using the `updateInterval` option when constructing a client. The default interval is 120 seconds.

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

## Supported Platforms

This SDK currently supports the following platforms/architectures:

- iOS arm64
- iOS Simulator arm64
- macOS arm64

## Migration Notes

### 1.1.0 -> 1.2.0

This section is for users who are migrating from a previous (1.1.0) version of the SDK.

- The `FliptClient` initializer now accepts an `environment` option. If not provided, the client will default to the `default` environment.
- `requestTimeout` and `updateInterval` are now `Duration` types instead of `Int`.

## Usage

```swift
import FliptClient

// Initialize the client
let client = try FliptClient(
    namespace: "default",
    url: "http://localhost:8080",
    authentication: .clientToken("your-token"),
    updateInterval: .seconds(120),
    fetchMode: .polling
)

// Evaluate a boolean flag
let boolResult = try client.evaluateBoolean(
    flagKey: "my-flag",
    entityId: "user-123",
    context: ["key": "value"]
)
print("Flag enabled: \(boolResult.enabled)")

// Evaluate a variant flag
let variantResult = try client.evaluateVariant(
    flagKey: "my-variant-flag",
    entityId: "user-123",
    context: ["key": "value"]
)
print("Variant key: \(variantResult.variantKey)")

// Don't forget to close the client when you're done
defer {
    client.close()
}
```

### Client Options

The `FliptClient` initializer accepts several options that can be used to configure the client. The available options are:

- `environment`: The environment (Flipt v2) to fetch flag state from. If not provided, the client will default to the `default` environment.
- `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
- `requestTimeout`: The timeout for total request time to the upstream Flipt instance. If not provided, the client will default to no timeout. Note: this only affects polling mode. Streaming mode will have no timeout set.
- `updateInterval`: The interval in which to fetch new flag state. If not provided, the client will default to 120 seconds.
- `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
- `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
- `fetchMode`: The fetch mode to use when fetching flag state. If not provided, the client will default to polling.
- `errorStrategy`: The error strategy to use when fetching flag state. If not provided, the client will default to `fail`. See the [Error Strategies](#error-strategies) section for more information.
- `snapshot`: The snapshot to use when fetching flag state. If not provided, the client will default to no snapshot. See the [Snapshot](#snapshotting) section for more information.

### Authentication

The `FliptClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### Error Strategies

The `FliptClient` supports the following error strategies:

- `fail`: The client will throw an error if the flag state cannot be fetched. This is the default behavior.
- `fallback`: The client will maintain the last known good state and use that state for evaluation in case of an error.

### Snapshotting

The client supports snapshotting of flag state as well as seeding the client with a snapshot for evaluation. This is helpful if you want to use the client in an environment where the Flipt server is not guaranteed to be available or reachable on startup.

To get the snapshot for the client, you can use the `getSnapshot` method. This returns a base64 encoded JSON string that represents the flag state for the client.

You can set the snapshot for the client using the `snapshot` option when constructing a client.

**Note:** You most likely will want to also set the `errorStrategy` to `fallback` when using snapshots. This will ensure that you wont get an error if the Flipt server is not available or reachable even on the initial fetch.

You also may want to store the snapshot in a local file so that you can use it to seed the client on startup.

> [!IMPORTANT]
> If the Flipt server becomes reachable after the setting the snapshot, the client will replace the snapshot with the new flag state from the Flipt server.

## Memory Management

The engine that is allocated on the Rust side to compute evaluations for flag state will not be properly deallocated unless you call the `close()` method on a `FliptClient` instance.

**Please be sure to do this to avoid leaking memory!**

```swift
defer {
    client.close()
}
```

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
