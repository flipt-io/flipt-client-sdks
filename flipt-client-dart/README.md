# Flipt Client Dart

[![pub package](https://img.shields.io/pub/v/flipt_client.svg)](https://pub.dev/packages/flipt_client)

The `flipt-client-dart` library contains the Dart source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

Add this to your package's `pubspec.yaml` file:

```yaml
dependencies:
  flipt_client: ^0.1.0
```

Then run `dart pub get` to install the package.

## How Does It Work?

The `flipt-client-dart` library is a wrapper around the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

All evaluation happens within the SDK, using the shared library built from the [flipt-engine-ffi](https://github.com/flipt-io/flipt-client-sdks/tree/main/flipt-engine-ffi) library.

Because the evaluation happens within the SDK, the SDKs can be used in environments where the Flipt server is not available or reachable after the initial data is fetched.

## Data Fetching

Upon instantiation, the `flipt-client-dart` library will fetch the flag state from the Flipt server and store it in memory. This means that the first time you use the SDK, it will make a request to the Flipt server.

### Polling (Default)

By default, the SDK will poll the Flipt server for new flag state at a regular interval. This interval can be configured using the `updateInterval` option when constructing a client. The default interval is 120 seconds.

### Streaming (Flipt Cloud Only)

[Flipt Cloud](https://flipt.io/cloud) users can use the `streaming` fetch method to stream flag state changes from the Flipt server to the SDK.

When in streaming mode, the SDK will connect to the Flipt server and open a persistent connection that will remain open until the client is closed. The SDK will then receive flag state changes in real-time.

## Supported Architectures

This SDK currently supports the following OSes/architectures:

- Linux x86_64
- Linux arm64
- MacOS x86_64
- MacOS arm64
- Windows x86_64

> [!IMPORTANT]
> This SDK currently only supports desktop platforms (MacOS, Linux, Windows).
> We are working on adding support for mobile platforms such as iOS and Android.

## Usage

```dart
import 'package:flipt_client/flipt_client.dart';

void main() {
  // Initialize the client
  final client = FliptEvaluationClient(
    'default',
    Options.withClientToken(
      'your-token',
      url: 'http://localhost:8080',
    ),
  );

  // Evaluate a variant flag
  final result = client.evaluateVariant(
    flagKey: 'flag1',
    entityId: 'someentity',
    context: {'fizz': 'buzz'},
  );

  print('Variant: ${result.variantKey}');

  // Evaluate a boolean flag
  final result = client.evaluateBoolean(
    flagKey: 'flag2',
    entityId: 'user123',
    context: {'key': 'value'},
  );

  print('Enabled: ${result.enabled}');

  // Don't forget to close the client when you're done
  client.close();
}
```

### Authentication

The `FliptEvaluationClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

## Development

To generate the JSON serialization code, run:

```bash
dart run build_runner build --delete-conflicting-outputs
```

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
