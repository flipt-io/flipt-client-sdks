# Flipt Client SDK for Android

The official Flipt client SDK for Android applications.

## Overview

The `flipt_client_android` package provides a Dart/Flutter SDK specifically optimized for Android platforms. It's a lightweight wrapper around the Flipt evaluation engine, enabling feature flag evaluation directly on Android devices.

## Installation

Add this to your package's `pubspec.yaml` file:

```yaml
dependencies:
  flipt_client_android: ^0.1.0
```

Then run:

```bash
flutter pub get
```

## Usage

```dart
import 'package:flipt_client_android/flipt_client_android.dart';

// Initialize the client
final client = FliptClient(
  options: Options(
    url: 'https://your-flipt-instance.com',
    updateInterval: 120,
  ),
);

// Evaluate a boolean flag
final boolResult = client.evaluateBoolean(
  flagKey: 'my-flag',
  entityId: 'user-123',
  context: {'role': 'admin'},
);
print('Flag enabled: ${boolResult.enabled}');

// Evaluate a variant flag
final variantResult = client.evaluateVariant(
  flagKey: 'my-variant-flag',
  entityId: 'user-123',
  context: {'plan': 'premium'},
);
print('Variant: ${variantResult.variantKey}');

// Don't forget to close the client when done
client.close();
```

## Features

- 🎯 **Zero-latency evaluation**: All flag evaluation happens locally on the device
- 📱 **Android optimized**: Specifically packaged for Android with minimal size (~12MB)
- 🔄 **Auto-polling**: Automatically fetches latest flag state at configurable intervals
- 🎨 **Type-safe**: Full Dart type safety with generated models

## Configuration Options

- `url`: Your Flipt instance URL
- `updateInterval`: How often to fetch updated flag state (in seconds, default: 120)
- `authentication`: Optional authentication configuration

## Platform Support

This package is specifically for Android. For other platforms, use:
- iOS: `flipt_client_ios`
- Desktop/Server: `flipt_client`

## Documentation

For more detailed documentation, visit:
- [Flipt Documentation](https://www.flipt.io/docs)
- [Client SDKs Documentation](https://www.flipt.io/docs/integration/client-sdks)

## License

This project is licensed under the MIT License.