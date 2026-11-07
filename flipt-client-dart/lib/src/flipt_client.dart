import 'dart:ffi';
import 'dart:convert';
import 'package:ffi/ffi.dart';
import 'package:flipt_client/src/models.dart';
import 'package:flipt_client/src/errors.dart';
import 'ffi/loader.dart';
import 'ffi/bindings.dart';
import 'state_updates.dart';

class FliptClient {
  late Pointer<Void> _engine;
  late FliptEngine _bindings;
  late final StateUpdates _stateUpdates;
  bool _closed = false;

  FliptClient({Options? options}) {
    final lib = loadFliptEngine();
    _bindings = FliptEngine(lib);
    _initializeEngine(options);
    _stateUpdates = StateUpdates(lib, _bindings, _engine);

    final onStateUpdate = options?.onStateUpdate;
    if (onStateUpdate != null) {
      try {
        addStateUpdateListener(onStateUpdate);
      } catch (_) {
        // The caller never gets a client to close, so tear the engine down here.
        close();
        rethrow;
      }
    }
  }

  /// Adds a listener that is called after each change to flag state.
  /// Listeners can be added at any time. Read the flags you care about once
  /// after adding a listener: it is called for changes applied after it was
  /// added, and may occasionally see one extra call.
  ///
  /// Listeners run on the isolate that created the client, in the order they
  /// were added, from its event loop. The blocking wait for changes runs on a
  /// helper isolate. Updates that arrive while listeners run are coalesced
  /// into a single follow-up call. A listener may evaluate flags, add or
  /// remove listeners, or call [close]. An exception thrown by one listener is
  /// written to stderr and does not stop the others.
  ///
  /// The client must be closed to stop its helper isolate.
  ///
  /// Throws [FliptError] if the client is closed, or [EvaluationError] if the
  /// engine cannot subscribe to updates.
  StateUpdateSubscription addStateUpdateListener(void Function() listener) {
    return _stateUpdates.add(listener);
  }

  void _checkOpen() {
    if (_closed) {
      throw FliptError('client is closed');
    }
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

  VariantEvaluationResponse evaluateVariant({
    required String flagKey,
    required String entityId,
    required Map<String, dynamic> context,
  }) {
    _checkOpen();
    if (flagKey.isEmpty) {
      throw ValidationError('flagKey must not be empty');
    }
    if (entityId.isEmpty) {
      throw ValidationError('entityId must not be empty');
    }

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
      throw EvaluationError(
          'Failed to evaluate variant: ${response.errorMessage}');
    }

    return response.result!;
  }

  BooleanEvaluationResponse evaluateBoolean({
    required String flagKey,
    required String entityId,
    required Map<String, dynamic> context,
  }) {
    _checkOpen();
    if (flagKey.isEmpty) {
      throw ValidationError('flagKey must not be empty');
    }
    if (entityId.isEmpty) {
      throw ValidationError('entityId must not be empty');
    }

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
      throw EvaluationError(
          'Failed to evaluate boolean: ${response.errorMessage}');
    }

    return response.result!;
  }

  BatchEvaluationResponse evaluateBatch(List<EvaluationRequest> requests) {
    _checkOpen();
    if (requests.isEmpty) {
      throw ValidationError('requests must not be empty');
    }
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
      throw EvaluationError(
          'Failed to evaluate batch: ${response.errorMessage}');
    }

    return response.result!;
  }

  List<Flag> listFlags() {
    _checkOpen();
    final resultPtr = _bindings.list_flags(_engine);
    final result = _pointerToString(resultPtr);
    _bindings.destroy_string(resultPtr);

    final response =
        FlagListResponse.fromJson(jsonDecode(result) as Map<String, dynamic>);

    if (response.status != Status.success) {
      throw EvaluationError('Failed to list flags: ${response.errorMessage}');
    }

    return response.toResult().result!;
  }

  String getSnapshot() {
    _checkOpen();
    final resultPtr = _bindings.get_snapshot(_engine);
    final result = _pointerToString(resultPtr);
    _bindings.destroy_string(resultPtr);
    return result;
  }

  /// Closes the client and releases the engine. No listener is called after
  /// this returns. Closing twice is a no-op.
  void close() {
    if (_closed) {
      return;
    }
    _closed = true;
    _stateUpdates.close();
    _bindings.destroy_engine(_engine);
  }
}
