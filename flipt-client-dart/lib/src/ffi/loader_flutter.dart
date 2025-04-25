// ignore_for_file: depend_on_referenced_packages
import 'dart:ffi';
import 'dart:io';
import 'package:path/path.dart' as path;
import 'loader.dart';

// Conditionally import Flutter services
import 'package:flutter/services.dart' show rootBundle deferred as flutter;

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

  // For desktop platforms, try standard path first, then assets
  final config = getPlatformConfig(getCurrentArchitecture());
  try {
    return DynamicLibrary.open(getPackagePath(config));
  } catch (e) {
    return _loadFromAssets(config);
  }
}

/// Loads the library from Flutter assets for desktop platforms
Future<DynamicLibrary> _loadFromAssets(LibraryConfig config) async {
  // Load Flutter services if needed
  await flutter.loadLibrary();
  
  final assetKey = config.assetPath;
  
  // Create a temporary directory to extract the library
  final tempDir = Directory.systemTemp.createTempSync('flipt_client');
  final tempPath = path.join(tempDir.path, config.fileName);
  
  try {
    // Copy the library from assets to temp directory
    final data = await flutter.rootBundle.load(assetKey);
    final bytes = data.buffer.asUint8List();
    await File(tempPath).writeAsBytes(bytes);
    
    // Make the library executable
    if (!Platform.isWindows) {
      await Process.run('chmod', ['+x', tempPath]);
    }
    
    return DynamicLibrary.open(tempPath);
  } catch (e) {
    throw UnsupportedError(
      'Failed to load native library from Flutter assets: $e\n'
      'Make sure the native binaries are included in your Flutter assets.',
    );
  }
}
