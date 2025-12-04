# Flipt Client Dart

[![flipt-client-dart](https://img.shields.io/pub/v/flipt_client.svg)](https://pub.dev/packages/flipt_client)

The `flipt-client-dart` library contains the Dart source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client.

## Installation

Add this to your package's `pubspec.yaml` file:

```yaml
dependencies:
  flipt_client: ^1.0.0
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

This SDK supports **mobile platforms only**:

- Android arm64
- iOS arm64 ⚠️ See [iOS Integration](#ios-integration)
- iOS arm64 (simulator) ⚠️ See [iOS Integration](#ios-integration)

> **Note**: Desktop platforms (Linux, macOS, Windows) are not supported due to pub.dev's 100MB package size limit.

## Migration Notes

### Pre-1.0.0 -> 1.0.0

This section is for users who are migrating from a previous (pre-1.0.0) version of the SDK.

- `FliptEvaluationClient` has been renamed to `FliptClient`.
- `Options` now accept `namespace` and `environment` as optional parameters.
- `requestTimeout` and `updateInterval` in `Options` are now `Duration` types instead of `int` (seconds). Update your code to use `Duration(seconds: ...)`.
- Desktop platforms (Linux, macOS, Windows) are no longer supported due to pub.dev's 100MB package size limit. Only iOS and Android are supported.

## Usage

```dart
import 'package:flipt_client/flipt_client.dart';

void main() {
  // Initialize the client
  final client = FliptClient(
    options: Options.withClientToken(
      'your-token',
      url: 'http://localhost:8080',
      requestTimeout: Duration(seconds: 10),
      updateInterval: Duration(minutes: 2),
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

The `FliptClient` initializer accepts several options that can be used to configure the client. The available options are:

- `environment`: The environment (Flipt v2) to fetch flag state from. If not provided, the client will default to the `default` environment.
- `namespace`: The namespace to fetch flag state from. If not provided, the client will default to the `default` namespace.
- `url`: The URL of the upstream Flipt instance. If not provided, the client will default to `http://localhost:8080`.
- `requestTimeout`: The timeout as a `Duration` for total request time to the upstream Flipt instance. If not provided, the client will default to no timeout. Note: this only affects polling mode. Streaming mode will have no timeout set.
- `updateInterval`: The interval as a `Duration` in which to fetch new flag state. If not provided, the client will default to `Duration(seconds: 120)`.
- `authentication`: The authentication strategy to use when communicating with the upstream Flipt instance. If not provided, the client will default to no authentication. See the [Authentication](#authentication) section for more information.
- `reference`: The [reference](https://docs.flipt.io/guides/user/using-references) to use when fetching flag state. If not provided, reference will not be used.
- `fetchMode`: The fetch mode to use when fetching flag state. If not provided, the client will default to polling.
- `errorStrategy`: The error strategy to use when fetching flag state. If not provided, the client will default to `fail`. See the [Error Strategies](#error-strategies) section for more information.
- `snapshot`: The initial snapshot to use when instantiating the client. See the [Snapshotting](#snapshotting) section for more information.

### Authentication

The `FliptClient` supports the following authentication strategies:

- No Authentication (default)
- [Client Token Authentication](https://docs.flipt.io/authentication/using-tokens)
- [JWT Authentication](https://docs.flipt.io/authentication/using-jwts)

### TLS Configuration

The `FliptClient` supports configuring TLS settings for secure connections to Flipt servers. This is useful when:

- Connecting to Flipt servers with self-signed certificates
- Using custom Certificate Authorities (CAs)
- Implementing mutual TLS authentication
- Testing with insecure connections (development only)

#### Basic TLS with Custom CA Certificate

```dart
import 'package:flipt_client/flipt_client.dart';

// Using a CA certificate file
final tlsConfig = TlsConfig.withCaCertFile('/path/to/ca.pem');

final client = FliptClient(
  options: Options.withClientToken(
    'your-token',
    url: 'https://flipt.example.com',
    tlsConfig: tlsConfig,
  ),
);
```

```dart
// Using CA certificate data directly
final caCertData = File('/path/to/ca.pem').readAsStringSync();
final tlsConfig = TlsConfig.withCaCertData(caCertData);

final client = FliptClient(
  options: Options.withClientToken(
    'your-token',
    url: 'https://flipt.example.com',
    tlsConfig: tlsConfig,
  ),
);
```

#### Mutual TLS Authentication

```dart
// Using certificate and key files
final tlsConfig = TlsConfig.withMutualTls('/path/to/client.pem', '/path/to/client.key');

final client = FliptClient(
  options: Options.withClientToken(
    'your-token',
    url: 'https://flipt.example.com',
    tlsConfig: tlsConfig,
  ),
);
```

```dart
// Using certificate and key data directly
final clientCertData = File('/path/to/client.pem').readAsStringSync();
final clientKeyData = File('/path/to/client.key').readAsStringSync();

final tlsConfig = TlsConfig.withMutualTlsData(clientCertData, clientKeyData);

final client = FliptClient(
  options: Options.withClientToken(
    'your-token',
    url: 'https://flipt.example.com',
    tlsConfig: tlsConfig,
  ),
);
```

#### Advanced TLS Configuration

```dart
// Full TLS configuration with all options
final tlsConfig = TlsConfig(
  caCertFile: '/path/to/ca.pem',
  clientCertFile: '/path/to/client.pem',
  clientKeyFile: '/path/to/client.key',
  insecureSkipVerify: false,
);

final client = FliptClient(
  options: Options.withClientToken(
    'your-token',
    url: 'https://flipt.example.com',
    tlsConfig: tlsConfig,
  ),
);
```

#### Development Mode (Insecure)

**⚠️ WARNING: Only use this in development environments!**

```dart
// Skip certificate verification (NOT for production)
final tlsConfig = TlsConfig.insecure();

final client = FliptClient(
  options: Options.withClientToken(
    'your-token',
    url: 'https://localhost:8443',
    tlsConfig: tlsConfig,
  ),
);
```

#### TLS Configuration Options

The `TlsConfig` class supports the following options:

- `caCertFile`: Path to custom CA certificate file (PEM format)
- `caCertData`: Raw CA certificate content (PEM format) - takes precedence over `caCertFile`
- `insecureSkipVerify`: Skip certificate verification (development only)
- `clientCertFile`: Client certificate file for mutual TLS (PEM format)
- `clientKeyFile`: Client private key file for mutual TLS (PEM format)
- `clientCertData`: Raw client certificate content (PEM format) - takes precedence over `clientCertFile`
- `clientKeyData`: Raw client private key content (PEM format) - takes precedence over `clientKeyFile`

> **Note**: When both file paths and data are provided, the data fields take precedence. For example, if both `caCertFile` and `caCertData` are set, `caCertData` will be used.

### Error Strategies

The client supports the following error strategies:

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

```console
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
