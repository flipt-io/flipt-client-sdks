import 'dart:ffi';
import 'dart:io';
import 'package:path/path.dart' as path;
import 'package:flutter/services.dart' show rootBundle;
import 'loader.dart';

/// Loads the Flipt engine dynamic library for Flutter applications.
///
/// This loader attempts to load the library from Flutter assets first,
/// falling back to the standard package loading mechanism if that fails.
Future<DynamicLibrary> loadFliptEngineFlutter() async {
  // Handle special cases first
  if (Platform.isAndroid) {
    return DynamicLibrary.open('libfliptengine.so');
  }

  if (Platform.isIOS) {
    return DynamicLibrary.process();
  }

  // Handle desktop platforms
  final arch = _getCurrentArchitecture();
  final config = _getPlatformConfig(arch);

  try {
    // Create a temporary directory to extract the library
    final tempDir = Directory.systemTemp.createTempSync('flipt_client');
    final tempFile = File(path.join(tempDir.path, config.fileName));

    // Read the asset as bytes
    final data = await rootBundle.load(config.assetPath);

    // Write to temporary file
    await tempFile.writeAsBytes(
      data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes),
      flush: true,
    );

    return DynamicLibrary.open(tempFile.path);
  } catch (_) {
    // If loading from assets fails, fall back to package loading
    return loadFliptEngine();
  }
}
