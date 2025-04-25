import 'dart:ffi';
import 'dart:io';
import 'package:path/path.dart' as path;

// Use conditional imports to handle Flutter vs pure Dart
import 'loader_io.dart'
    if (dart.library.html) 'loader_web.dart'
    if (dart.library.flutter) 'loader_flutter.dart';

/// Represents the supported CPU architectures
enum Architecture {
  x86_64,
  arm64,
}

/// Configuration for platform-specific library details
class LibraryConfig {
  final String directory;
  final String fileName;

  const LibraryConfig(this.directory, this.fileName);

  String get assetPath => 'native/$directory/$fileName';
}

/// Gets the current CPU architecture
Architecture getCurrentArchitecture() {
  return Platform.version.contains('arm64')
      ? Architecture.arm64
      : Architecture.x86_64;
}

/// Gets the platform-specific library configuration
LibraryConfig getPlatformConfig(Architecture arch) {
  // We only need platform configs for desktop platforms
  // Android and iOS are handled directly in their respective loaders
  if (Platform.isMacOS) {
    final dir = arch == Architecture.arm64 ? 'darwin_aarch64' : 'darwin_x86_64';
    return LibraryConfig(dir, 'libfliptengine.dylib');
  }

  if (Platform.isLinux) {
    final dir = arch == Architecture.arm64 ? 'linux_aarch64' : 'linux_x86_64';
    return LibraryConfig(dir, 'libfliptengine.so');
  }

  if (Platform.isWindows) {
    return const LibraryConfig('windows_x86_64', 'fliptengine.dll');
  }

  throw UnsupportedError('Unsupported platform: ${Platform.operatingSystem}');
}

/// Gets the path to the binary relative to the package root
String getPackagePath(LibraryConfig config) {
  final libraryPath = path.join(
    Directory.current.path,
    'native',
    config.directory,
    config.fileName,
  );

  if (!File(libraryPath).existsSync()) {
    throw UnsupportedError(
      'Could not find native library at: $libraryPath\n'
      'Make sure the native binaries are present in the binaries directory.',
    );
  }

  return libraryPath;
}

/// Handles loading for mobile platforms (Android and iOS)
DynamicLibrary? loadMobilePlatform() {
  if (Platform.isAndroid) {
    // Android libraries are loaded from the app's library directory
    return DynamicLibrary.open('libfliptengine.so');
  }

  // if (Platform.isIOS) {
  //   // iOS libraries are statically linked into the binary
  //   return DynamicLibrary.executable();
  // }

  return null;
}

/// Loads the Flipt engine dynamic library for the current platform.
/// This is the main entry point for loading the native library.
/// The implementation is selected at compile time based on the platform.
DynamicLibrary loadFliptEngine() => loadPlatformDependentFliptEngine();
