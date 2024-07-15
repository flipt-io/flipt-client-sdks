import 'dart:convert';
import 'dart:ffi' as ffi;
import 'dart:io' show Directory;
import 'package:ffi/ffi.dart';
import 'package:flipt_client_flutter/flipt_client.dart';
import 'package:path/path.dart' as path;

String _getLibraryPath() {
  var libraryPath = '';
  switch (ffi.Abi.current()) {
    case ffi.Abi.macosArm64:
      libraryPath = 'libfliptengine.dylib';
      break;
    default:
      libraryPath = 'libfliptengine.so';
  }

  final currentDirectory = Directory.current.path;
  return path.join(currentDirectory, 'ext', libraryPath);
}

// C function:
// void *initialize_engine(const char *namespace_, const char *opts);
typedef InitializeEngineFunc = ffi.Pointer<ffi.Void> Function(
    ffi.Pointer<Utf8> namespace, ffi.Pointer<Utf8> opts);

ffi.Pointer<ffi.Void> initializeEngine(
    ffi.Pointer<Utf8> namespace, ffi.Pointer<Utf8> opts) {
  final dylib = ffi.DynamicLibrary.open(_getLibraryPath());
  final initializeEngineFFI =
      dylib.lookupFunction<InitializeEngineFunc, InitializeEngineFunc>(
          'initialize_engine');
  return initializeEngineFFI(namespace, opts);
}

// C function:
// void destroy_engine(void *engine);
typedef DestroyEngineFunc = ffi.Void Function(ffi.Pointer<ffi.Void> engine);

void destroyEngine(ffi.Pointer<ffi.Void> engine) {
  final dylib = ffi.DynamicLibrary.open(_getLibraryPath());
  final destroyEngineFFI = dylib.lookupFunction<DestroyEngineFunc,
      void Function(ffi.Pointer<ffi.Void>)>('destroy_engine');
  destroyEngineFFI(engine);
}

// C function:
// const char *evaluate_variant(void *engine_ptr, const char *evaluation_request);
typedef EvaluateVariantFunc = ffi.Pointer<Utf8> Function(
    ffi.Pointer<ffi.Void> engine, ffi.Pointer<Utf8> evaluationRequest);

String _evaluateVariant(
    ffi.Pointer<ffi.Void> engine, String evaluationRequest) {
  final dylib = ffi.DynamicLibrary.open(_getLibraryPath());
  final evaluateVariantFFI = dylib.lookupFunction<
      EvaluateVariantFunc,
      ffi.Pointer<Utf8> Function(
          ffi.Pointer<ffi.Void>, ffi.Pointer<Utf8>)>('evaluate_variant');
  final evaluationRequestNative = evaluationRequest.toNativeUtf8();
  final evaluateVariantResponse =
      evaluateVariantFFI(engine, evaluationRequestNative).toDartString();
  calloc.free(evaluationRequestNative);
  return evaluateVariantResponse;
}

/// FliptEvaluationClient is a Flipt client.
class FliptEvaluationClient {
  late final String? _namespace;
  late final ffi.Pointer<ffi.Void> _engine;

  /// Creates a new client.
  /// [namespace] is the namespace of the client.
  /// [url] is the URL of the Flipt server.
  /// [ref] is the optional reference to evaluate.
  /// [updateInterval] is the interval to update the state from the server.
  FliptEvaluationClient({
    namespace = "default",
    url = "http://localhost:8080",
    ref,
    updateInterval = const Duration(seconds: 120),
  }) {
    _namespace = namespace;

    var opts = {
      'url': url,
      'reference': ref,
      'update_interval': updateInterval.inSeconds,
    };

    var namespaceNative = _namespace!.toNativeUtf8();
    var optsJsonNative = jsonEncode(opts).toNativeUtf8();

    _engine = initializeEngine(namespaceNative, optsJsonNative);

    calloc.free(namespaceNative);
    calloc.free(optsJsonNative);
  }

  /// Destroys the client.
  /// This method should be called when the client is no longer needed.
  void destroy() {
    destroyEngine(_engine);
  }

  VariantResult evaluateVariant(
      {required String flagKey,
      String entityId = '',
      Map<String, String>? context}) {
    var request = EvaluationRequest()
      ..flagKey = flagKey
      ..entityId = entityId
      ..context = context;

    var requestJson = jsonEncode(request);
    var responseJson = _evaluateVariant(_engine, requestJson);

    var responseMap = jsonDecode(responseJson);
    return VariantResult.fromJson(responseMap);
  }
}
