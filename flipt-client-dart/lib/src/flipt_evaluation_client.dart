import 'dart:ffi';
import 'dart:io';
import 'dart:convert';
import 'package:ffi/ffi.dart';
import 'package:flipt_client/src/models.dart';
import 'package:path/path.dart' as path;

typedef InitializeEngineNative = Pointer<Void> Function(
    Pointer<Utf8> namespace, Pointer<Utf8> opts);
typedef InitializeEngineDart = Pointer<Void> Function(
    Pointer<Utf8> namespace, Pointer<Utf8> opts);

typedef EvaluateVariantNative = Pointer<Utf8> Function(
    Pointer<Void> engine, Pointer<Utf8> evaluationRequest);
typedef EvaluateVariantDart = Pointer<Utf8> Function(
    Pointer<Void> engine, Pointer<Utf8> evaluationRequest);

typedef EvaluateBooleanNative = Pointer<Utf8> Function(
    Pointer<Void> engine, Pointer<Utf8> evaluationRequest);
typedef EvaluateBooleanDart = Pointer<Utf8> Function(
    Pointer<Void> engine, Pointer<Utf8> evaluationRequest);

typedef EvaluateBatchNative = Pointer<Utf8> Function(
    Pointer<Void> engine, Pointer<Utf8> evaluationRequests);
typedef EvaluateBatchDart = Pointer<Utf8> Function(
    Pointer<Void> engine, Pointer<Utf8> evaluationRequests);

typedef DestroyEngineNative = Void Function(Pointer<Void> engine);
typedef DestroyEngineDart = void Function(Pointer<Void> engine);

typedef DestroyStringNative = Void Function(Pointer<Utf8> str);
typedef DestroyStringDart = void Function(Pointer<Utf8> str);

class FliptEvaluationClient {
  late Pointer<Void> _engine;
  late DynamicLibrary _lib;

  FliptEvaluationClient({String namespace = "default", Options? options}) {
    _lib = DynamicLibrary.open(_getLibraryPath());
    _initializeEngine(namespace, options);
  }

  String _getLibraryPath() {
    final String libraryName;
    final String platformDir;

    switch (Platform.operatingSystem) {
      case 'linux':
        libraryName = 'libfliptengine.so';
        platformDir =
            Platform.version.contains('arm64') ? 'linux_arm64' : 'linux_x86_64';
        break;
      case 'macos':
        libraryName = 'libfliptengine.dylib';
        platformDir = Platform.version.contains('arm64')
            ? 'darwin_arm64'
            : 'darwin_x86_64';
        break;
      default:
        throw UnsupportedError(
            'Unsupported platform: ${Platform.operatingSystem}');
    }

    return path.join(
      Directory.current.path,
      'lib',
      'src',
      'ffi',
      platformDir,
      libraryName,
    );
  }

  void _initializeEngine(String namespace, Options? options) {
    final initializeEngine =
        _lib.lookupFunction<InitializeEngineNative, InitializeEngineDart>(
            'initialize_engine');

    final namespaceUtf8 = namespace.toNativeUtf8();
    final optsJson = jsonEncode(options?.toJson() ?? {});
    final optsUtf8 = optsJson.toNativeUtf8();

    try {
      _engine = initializeEngine(namespaceUtf8, optsUtf8);
    } finally {
      calloc.free(namespaceUtf8);
      calloc.free(optsUtf8);
    }
  }

  Result<VariantEvaluationResponse> evaluateVariant(
      {required String flagKey,
      required String entityId,
      required Map<String, dynamic> context}) {
    final evaluateVariant =
        _lib.lookupFunction<EvaluateVariantNative, EvaluateVariantDart>(
            'evaluate_variant');

    final request = EvaluationRequest(
      flagKey: flagKey,
      entityId: entityId,
      context: context,
    );

    final requestJson = jsonEncode(request.toJson());
    final requestUtf8 = requestJson.toNativeUtf8();

    final resultPtr = evaluateVariant(_engine, requestUtf8);
    final result = resultPtr.toDartString();
    _destroyString(resultPtr);

    try {
      final response = Result<VariantEvaluationResponse>.fromJson(
        jsonDecode(result) as Map<String, dynamic>,
        (json) =>
            VariantEvaluationResponse.fromJson(json as Map<String, dynamic>),
      );

      if (response.status != EvaluationStatus.success) {
        throw Exception('Failed to evaluate variant: ${response.errorMessage}');
      }

      return response;
    } finally {
      calloc.free(requestUtf8);
    }
  }

  Result<BooleanEvaluationResponse> evaluateBoolean(
      {required String flagKey,
      required String entityId,
      required Map<String, dynamic> context}) {
    final evaluateBoolean =
        _lib.lookupFunction<EvaluateBooleanNative, EvaluateBooleanDart>(
            'evaluate_boolean');

    final request = EvaluationRequest(
      flagKey: flagKey,
      entityId: entityId,
      context: context,
    );

    final requestJson = jsonEncode(request.toJson());
    final requestUtf8 = requestJson.toNativeUtf8();

    try {
      final resultPtr = evaluateBoolean(_engine, requestUtf8);
      final result = resultPtr.toDartString();
      _destroyString(resultPtr);

      final response = Result<BooleanEvaluationResponse>.fromJson(
        jsonDecode(result) as Map<String, dynamic>,
        (json) =>
            BooleanEvaluationResponse.fromJson(json as Map<String, dynamic>),
      );

      if (response.status != EvaluationStatus.success) {
        throw Exception('Failed to evaluate boolean: ${response.errorMessage}');
      }

      return response;
    } finally {
      calloc.free(requestUtf8);
    }
  }

  Result<BatchEvaluationResponse> evaluateBatch(
      List<EvaluationRequest> requests) {
    final evaluateBatch =
        _lib.lookupFunction<EvaluateBatchNative, EvaluateBatchDart>(
            'evaluate_batch');

    final requestsJson = jsonEncode(requests.map((r) => r.toJson()).toList());
    final requestsUtf8 = requestsJson.toNativeUtf8();

    try {
      final resultPtr = evaluateBatch(_engine, requestsUtf8);
      final result = resultPtr.toDartString();
      _destroyString(resultPtr);

      final response = Result<BatchEvaluationResponse>.fromJson(
        jsonDecode(result) as Map<String, dynamic>,
        (json) =>
            BatchEvaluationResponse.fromJson(json as Map<String, dynamic>),
      );

      return response;
    } finally {
      calloc.free(requestsUtf8);
    }
  }

  void _destroyString(Pointer<Utf8> str) {
    final destroyString =
        _lib.lookupFunction<DestroyStringNative, DestroyStringDart>(
            'destroy_string');
    destroyString(str);
  }

  void close() {
    final destroyEngine =
        _lib.lookupFunction<DestroyEngineNative, DestroyEngineDart>(
            'destroy_engine');
    destroyEngine(_engine);
  }
}
