import 'dart:ffi';
import 'dart:io';
import 'dart:convert';
import 'package:ffi/ffi.dart';
import 'package:flipt_client/src/models.dart';
import 'package:path/path.dart' as path;
import 'ffi/loader.dart';
import 'ffi/bindings.dart';

class FliptEvaluationClient {
  late Pointer<Void> _engine;
  late FliptEngine _bindings;

  FliptEvaluationClient({Options? options}) {
    final lib = loadFliptEngine();
    _bindings = FliptEngine(lib);
    _initializeEngine(options);
  }

  void _initializeEngine(Options? options) {
    final optsJson = jsonEncode(options?.toJson() ?? {});
    final optsUtf8 = optsJson.toNativeUtf8();

    try {
      _engine = _bindings.initialize_engine(
        optsUtf8.cast<Char>(),
      );
    } finally {
      calloc.free(optsUtf8);
    }
  }

  String _pointerToString(Pointer<Char> ptr) {
    return ptr.cast<Utf8>().toDartString();
  }

  VariantEvaluationResponse evaluateVariant(
      {required String flagKey,
      required String entityId,
      required Map<String, dynamic> context}) {
    final request = EvaluationRequest(
      flagKey: flagKey,
      entityId: entityId,
      context: context,
    );

    final requestJson = jsonEncode(request.toJson());
    final requestUtf8 = requestJson.toNativeUtf8();

    final resultPtr =
        _bindings.evaluate_variant(_engine, requestUtf8.cast<Char>());
    final result = _pointerToString(resultPtr);
    _bindings.destroy_string(resultPtr);
    calloc.free(requestUtf8);

    final response = Result<VariantEvaluationResponse>.fromJson(
      jsonDecode(result) as Map<String, dynamic>,
      (json) =>
          VariantEvaluationResponse.fromJson(json as Map<String, dynamic>),
    );

    if (response.status != Status.success) {
      throw Exception('Failed to evaluate variant: ${response.errorMessage}');
    }

    return response.result!;
  }

  BooleanEvaluationResponse evaluateBoolean(
      {required String flagKey,
      required String entityId,
      required Map<String, dynamic> context}) {
    final request = EvaluationRequest(
      flagKey: flagKey,
      entityId: entityId,
      context: context,
    );

    final requestJson = jsonEncode(request.toJson());
    final requestUtf8 = requestJson.toNativeUtf8();

    final resultPtr =
        _bindings.evaluate_boolean(_engine, requestUtf8.cast<Char>());
    final result = _pointerToString(resultPtr);
    _bindings.destroy_string(resultPtr);
    calloc.free(requestUtf8);

    final response = Result<BooleanEvaluationResponse>.fromJson(
      jsonDecode(result) as Map<String, dynamic>,
      (json) =>
          BooleanEvaluationResponse.fromJson(json as Map<String, dynamic>),
    );

    if (response.status != Status.success) {
      throw Exception('Failed to evaluate boolean: ${response.errorMessage}');
    }

    return response.result!;
  }

  BatchEvaluationResponse evaluateBatch(List<EvaluationRequest> requests) {
    final requestsJson = jsonEncode(requests.map((r) => r.toJson()).toList());
    final requestsUtf8 = requestsJson.toNativeUtf8();

    final resultPtr =
        _bindings.evaluate_batch(_engine, requestsUtf8.cast<Char>());
    final result = _pointerToString(resultPtr);
    _bindings.destroy_string(resultPtr);
    calloc.free(requestsUtf8);

    final response = Result<BatchEvaluationResponse>.fromJson(
      jsonDecode(result) as Map<String, dynamic>,
      (json) => BatchEvaluationResponse.fromJson(json as Map<String, dynamic>),
    );

    if (response.status != Status.success) {
      throw Exception('Failed to evaluate batch: ${response.errorMessage}');
    }

    return response.result!;
  }

  List<Flag> listFlags() {
    final resultPtr = _bindings.list_flags(_engine);
    final result = _pointerToString(resultPtr);
    _bindings.destroy_string(resultPtr);

    final response =
        FlagListResponse.fromJson(jsonDecode(result) as Map<String, dynamic>);

    if (response.status != Status.success) {
      throw Exception('Failed to list flags: ${response.errorMessage}');
    }

    return response.toResult().result!;
  }

  String getSnapshot() {
    final resultPtr = _bindings.get_snapshot(_engine);
    final result = _pointerToString(resultPtr);
    _bindings.destroy_string(resultPtr);
    return result;
  }

  void close() {
    _bindings.destroy_engine(_engine);
  }
}
