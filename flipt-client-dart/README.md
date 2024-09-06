# Flipt Client Dart

[![pub package](https://img.shields.io/pub/v/flipt_client.svg)](https://pub.dev/packages/flipt_client)

The `flipt-client-dart` library contains the Dart source code for the Flipt [client-side evaluation](https://www.flipt.io/docs/integration/client) client. T

## Installation

Add this to your package's `pubspec.yaml` file:

```yaml
dependencies:
  flipt_client: ^0.1.0
```

Then run `dart pub get` to install the package.

## Supported Architectures

This SDK currently supports the following OSes/architectures:

- Linux x86_64
- Linux arm64
- MacOS x86_64
- MacOS arm64

## Usage

```dart
import 'package:flipt_client/flipt_client.dart';

void main() {
  // Initialize the client
  final client = FliptEvaluationClient(
    'default',
    EngineOpts(
      url: 'http://localhost:8080',
      authentication: ClientTokenAuthentication('your-token'),
    ),
  );

  // Evaluate a variant flag
  final variantResult = client.evaluateVariant(
    'flag1',
    'user123',
    {'key': 'value'},
  );
  print('Variant: ${variantResult.variantKey}');

  // Evaluate a boolean flag
  final booleanResult = client.evaluateBoolean(
    'flag2',
    'user123',
    {'key': 'value'},
  );
  print('Enabled: ${booleanResult.enabled}');

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