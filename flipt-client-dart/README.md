# Flipt Client Dart

[![flipt-client-dart](https://img.shields.io/pub/v/flipt_client.svg)](https://pub.dev/packages/flipt_client)

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
- Android arm64
- iOS arm64 ⚠️ See [iOS Integration](#ios-integration)
- iOS arm64 (simulator) ⚠️ See [iOS Integration](#ios-integration)

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

### Client Options

The `FliptEvaluationClient` initializer accepts several options that can be used to configure the client. The available options are:

- `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
- `requestTimeout`: The timeout (in seconds) for total request time to the upstream Flipt instance. If not provided, the client will default to no timeout. Note: this only affects polling mode. Streaming mode will have no timeout set.
- `updateInterval`: The interval (in seconds) in which to fetch new flag state. If not provided, the client will default to 120 seconds.
- `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
- `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
- `fetchMode`: The fetch mode to use when fetching flag state. If not provided, the client will default to polling.
- `errorStrategy`: The error strategy to use when fetching flag state. If not provided, the client will default to `fail`. See the [Error Strategies](#error-strategies) section for more information.

### Authentication

The `FliptEvaluationClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### Error Strategies

The client supports the following error strategies:

- `fail`: The client will throw an error if the flag state cannot be fetched. This is the default behavior.
- `fallback`: The client will maintain the last known good state and use that state for evaluation in case of an error.

## iOS Integration 📱

The `flipt-client-dart` library can be used in a Flutter iOS app but requires a few additional steps to integrate.

We're working on making this easier in the future.

### Podfile

Open your app's `ios/Podfile` and comment out the following lines in the `target 'Runner' do` block:

```ruby
# use_frameworks!
# use_modular_headers!
```

### Troubleshooting

#### Missing x86_64

If you see errors about missing x86_64, add the following to the `post_install` block of your `Podfile`:

```ruby
post_install do |installer|
  installer.pods_project.targets.each do |target|
    flutter_additional_ios_build_settings(target)
    # Exclude x86_64 for simulator builds (Apple Silicon only)
    target.build_configurations.each do |config|
      config.build_settings['EXCLUDED_ARCHS[sdk=iphonesimulator*]'] = 'x86_64'
    end
  end
end
```

You may also need to add the following `User Defined Setting` to your Xcode project. Do this in the `Build Settings` tab for your project, not the target!

```
VALID_ARCHS = arm64
```

![User Defined Settings](/.github/images/flutter-ios-user-defined-settings.png)

## Development

### Install Dependencies

To install the dependencies, run:

```bash
dart pub get
```

### Generate JSON Serialization Code

To generate the JSON serialization code (found in `lib/src/models.g.dart`), run:

```bash
dart run build_runner build --delete-conflicting-outputs
```

### Generate FFI Bindings

To generate the FFI bindings (found in `lib/ffi/bindings.dart`), run:

```bash
cp ../flipt-engine-ffi/include/flipt_engine.h ffi/flipt_engine.h
dart run ffigen --config ffigen.yaml
```

## Contributing

Contributions are welcome! Please feel free to open an issue or submit a Pull Request.

## License

This project is licensed under the [MIT License](LICENSE).
