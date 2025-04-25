import 'dart:ffi';
import 'dart:io';
import 'package:path/path.dart' as path;
import 'loader.dart' as common;

/// Platform-specific implementation for pure Dart IO platforms
DynamicLibrary loadPlatformDependentFliptEngine() {
  // Handle special cases first
  if (Platform.isAndroid) {
    // Android libraries are handled via jniLibs
    return DynamicLibrary.open('libfliptengine.so');
  }

  if (Platform.isIOS) {
    // iOS libraries are statically linked
    return DynamicLibrary.process();
  }

  // Handle desktop platforms
  final arch = common.getCurrentArchitecture();
  final config = common.getPlatformConfig(arch);
  final libraryPath = common.getPackagePath(config);

  if (!File(libraryPath).existsSync()) {
    throw UnsupportedError(
      'Could not find native library at: $libraryPath\n'
      'Make sure the flipt-client-dart package is properly installed with its native binaries.',
    );
  }

  return DynamicLibrary.open(libraryPath);
}
