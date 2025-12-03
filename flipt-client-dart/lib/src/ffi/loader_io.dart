import 'dart:ffi';
import 'dart:io';
import 'loader.dart' as common;

/// Platform-specific implementation for pure Dart IO platforms
/// Only supports mobile platforms (Android, iOS)
DynamicLibrary loadPlatformDependentFliptEngine() {
  // Try mobile platforms first
  final mobileLib = common.loadMobilePlatform();
  if (mobileLib != null) {
    return mobileLib;
  }

  // If we get here, we're on an unsupported platform
  // getPlatformConfig will throw
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
