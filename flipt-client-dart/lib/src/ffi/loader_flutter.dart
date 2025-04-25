// ignore_for_file: depend_on_referenced_packages
import 'dart:ffi';
import 'dart:io';
import 'package:path/path.dart' as path;
import 'loader.dart';

/// Loads the Flipt engine dynamic library for Flutter platforms.
/// This implementation handles loading for all Flutter platforms:
/// - Android: Loads from app-specific library directory
/// - iOS: Uses statically linked library
/// - Desktop: Loads from assets or standard paths
DynamicLibrary loadPlatformDependentFliptEngine() {
  // Try mobile platforms first
  final mobileLib = loadMobilePlatform();
  if (mobileLib != null) {
    return mobileLib;
  }

  // For desktop platforms, try standard path first
  final config = getPlatformConfig(getCurrentArchitecture());
  try {
    return DynamicLibrary.open(getPackagePath(config));
  } catch (e) {
    // If standard path fails, fall back to synchronous asset loading
    return _loadFromAssetsSync(config);
  }
}

/// Loads the library from Flutter assets for desktop platforms
/// This is a synchronous version that loads from a pre-extracted location
DynamicLibrary _loadFromAssetsSync(LibraryConfig config) {
  // Create a temporary directory to extract the library
  final tempDir = Directory.systemTemp.createTempSync('flipt_client');
  final tempPath = path.join(tempDir.path, config.fileName);

  if (!File(tempPath).existsSync()) {
    throw UnsupportedError(
      'Could not find native library in assets.\n'
      'Make sure the native binaries are included in your Flutter assets.',
    );
  }

  // Make the library executable if needed
  if (!Platform.isWindows) {
    Process.runSync('chmod', ['+x', tempPath]);
  }

  return DynamicLibrary.open(tempPath);
}
