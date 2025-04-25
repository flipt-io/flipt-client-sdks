import 'dart:ffi';
import 'dart:io';
import 'package:path/path.dart' as path;
import 'package:flutter/services.dart' show rootBundle;
import 'loader.dart' as common;
import 'loader_io.dart' as io;

/// Platform-specific implementation for Flutter platforms
DynamicLibrary loadPlatformDependentFliptEngine() {
  // Handle mobile platforms the same way
  if (Platform.isAndroid || Platform.isIOS) {
    return io.loadPlatformDependentFliptEngine();
  }

  // For desktop platforms, try loading from assets first
  final arch = common.getCurrentArchitecture();
  final config = common.getPlatformConfig(arch);

  try {
    // Create a temporary directory to extract the library
    final tempDir = Directory.systemTemp.createTempSync('flipt_client');
    final tempFile = File(path.join(tempDir.path, config.fileName));

    // Try to load from Flutter assets synchronously
    final data = rootBundle.load(config.assetPath).then((byteData) {
      return tempFile
          .writeAsBytes(
            byteData.buffer
                .asUint8List(byteData.offsetInBytes, byteData.lengthInBytes),
            flush: true,
          )
          .then((_) => tempFile);
    });

    // If we can't load from assets, fall back to package loading
    return data
        .then((file) => DynamicLibrary.open(file.path))
        .catchError((_) => io.loadPlatformDependentFliptEngine())
        // Force synchronous completion
        .sync;
  } catch (_) {
    // If anything goes wrong, fall back to package loading
    return io.loadPlatformDependentFliptEngine();
  }
}
