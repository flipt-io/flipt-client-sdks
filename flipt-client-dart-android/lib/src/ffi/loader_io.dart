import 'dart:ffi';
import 'dart:io';
import 'package:path/path.dart' as path;
import 'loader.dart' as common;

/// Platform-specific implementation for pure Dart IO platforms
DynamicLibrary loadPlatformDependentFliptEngine() {
  // Try mobile platforms first
  final mobileLib = common.loadMobilePlatform();
  if (mobileLib != null) {
    return mobileLib;
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
