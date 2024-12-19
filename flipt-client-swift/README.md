# Flipt Client Swift

[![Client tag](https://img.shields.io/github/v/tag/flipt-io/flipt-client-swift?filter=v*&label=flipt-client-swift)](https://github.com/flipt-io/flipt-client-swift)

The `flipt-client-swift` library contains the Swift source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

### Swift Package Manager

Add the following to your `Package.swift` file:

```swift
dependencies: [
    .package(url: "https://github.com/flipt-io/flipt-client-swift.git", from: "0.0.x")
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

### Streaming (Flipt Cloud Only)

[Flipt Cloud](https://flipt.io/cloud) users can use the `streaming` fetch method to stream flag state changes from the Flipt server to the SDK.

When in streaming mode, the SDK will connect to the Flipt server and open a persistent connection that will remain open until the client is closed. The SDK will then receive flag state changes in real-time.

## Supported Platforms

This SDK currently supports the following platforms/architectures:

- iOS arm64
- iOS Simulator arm64
- macOS arm64

## Usage

```swift
import FliptClient

// Initialize the client
let client = try FliptClient(
    namespace: "default",
    url: "http://localhost:8080",
    authentication: .clientToken("your-token"),
    updateInterval: 120,
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

### Authentication

The `FliptClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

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
