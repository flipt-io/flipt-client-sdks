// ignore_for_file: depend_on_referenced_packages
import 'dart:ffi';
import 'loader.dart';

/// Loads the Flipt engine dynamic library for Flutter platforms.
/// This implementation supports mobile platforms only:
/// - Android: Loads from app-specific library directory
/// - iOS: Uses statically linked library
///
/// Desktop platforms (macOS, Linux, Windows) are not supported in this package.
DynamicLibrary loadPlatformDependentFliptEngine() {
  // Try mobile platforms first
  final mobileLib = loadMobilePlatform();
  if (mobileLib != null) {
    return mobileLib;
  }

  // If we get here, we're on an unsupported platform
  // getPlatformConfig will throw
  final config = getPlatformConfig(getCurrentArchitecture());
  return DynamicLibrary.open(getPackagePath(config));
}
