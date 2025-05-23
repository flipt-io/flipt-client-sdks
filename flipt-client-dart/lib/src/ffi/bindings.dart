// AUTO GENERATED FILE, DO NOT EDIT.
//
// Generated by `package:ffigen`.
// ignore_for_file: type=lint
import 'dart:ffi' as ffi;

/// Auto-generated FFI bindings for flipt_engine
class FliptEngine {
  /// Holds the symbol lookup function.
  final ffi.Pointer<T> Function<T extends ffi.NativeType>(String symbolName)
      _lookup;

  /// The symbols are looked up in [dynamicLibrary].
  FliptEngine(ffi.DynamicLibrary dynamicLibrary)
      : _lookup = dynamicLibrary.lookup;

  /// The symbols are looked up with [lookup].
  FliptEngine.fromLookup(
      ffi.Pointer<T> Function<T extends ffi.NativeType>(String symbolName)
          lookup)
      : _lookup = lookup;

  /// # Safety
  ///
  /// This function will initialize an Engine and return a pointer back to the caller.
  ffi.Pointer<ffi.Void> initialize_engine(
    ffi.Pointer<ffi.Char> namespace_,
    ffi.Pointer<ffi.Char> opts,
  ) {
    return _initialize_engine(
      namespace_,
      opts,
    );
  }

  late final _initialize_enginePtr = _lookup<
      ffi.NativeFunction<
          ffi.Pointer<ffi.Void> Function(ffi.Pointer<ffi.Char>,
              ffi.Pointer<ffi.Char>)>>('initialize_engine');
  late final _initialize_engine = _initialize_enginePtr.asFunction<
      ffi.Pointer<ffi.Void> Function(
          ffi.Pointer<ffi.Char>, ffi.Pointer<ffi.Char>)>();

  /// # Safety
  ///
  /// This function will take in a pointer to the engine and return a variant evaluation response.
  ffi.Pointer<ffi.Char> evaluate_variant(
    ffi.Pointer<ffi.Void> engine_ptr,
    ffi.Pointer<ffi.Char> evaluation_request,
  ) {
    return _evaluate_variant(
      engine_ptr,
      evaluation_request,
    );
  }

  late final _evaluate_variantPtr = _lookup<
      ffi.NativeFunction<
          ffi.Pointer<ffi.Char> Function(ffi.Pointer<ffi.Void>,
              ffi.Pointer<ffi.Char>)>>('evaluate_variant');
  late final _evaluate_variant = _evaluate_variantPtr.asFunction<
      ffi.Pointer<ffi.Char> Function(
          ffi.Pointer<ffi.Void>, ffi.Pointer<ffi.Char>)>();

  /// # Safety
  ///
  /// This function will take in a pointer to the engine and return a boolean evaluation response.
  ffi.Pointer<ffi.Char> evaluate_boolean(
    ffi.Pointer<ffi.Void> engine_ptr,
    ffi.Pointer<ffi.Char> evaluation_request,
  ) {
    return _evaluate_boolean(
      engine_ptr,
      evaluation_request,
    );
  }

  late final _evaluate_booleanPtr = _lookup<
      ffi.NativeFunction<
          ffi.Pointer<ffi.Char> Function(ffi.Pointer<ffi.Void>,
              ffi.Pointer<ffi.Char>)>>('evaluate_boolean');
  late final _evaluate_boolean = _evaluate_booleanPtr.asFunction<
      ffi.Pointer<ffi.Char> Function(
          ffi.Pointer<ffi.Void>, ffi.Pointer<ffi.Char>)>();

  /// # Safety
  ///
  /// This function will take in a pointer to the engine and return a batch evaluation response.
  ffi.Pointer<ffi.Char> evaluate_batch(
    ffi.Pointer<ffi.Void> engine_ptr,
    ffi.Pointer<ffi.Char> batch_evaluation_request,
  ) {
    return _evaluate_batch(
      engine_ptr,
      batch_evaluation_request,
    );
  }

  late final _evaluate_batchPtr = _lookup<
      ffi.NativeFunction<
          ffi.Pointer<ffi.Char> Function(
              ffi.Pointer<ffi.Void>, ffi.Pointer<ffi.Char>)>>('evaluate_batch');
  late final _evaluate_batch = _evaluate_batchPtr.asFunction<
      ffi.Pointer<ffi.Char> Function(
          ffi.Pointer<ffi.Void>, ffi.Pointer<ffi.Char>)>();

  /// # Safety
  ///
  /// This function will take in a pointer to the engine and return a list of flags for the given namespace.
  ffi.Pointer<ffi.Char> list_flags(
    ffi.Pointer<ffi.Void> engine_ptr,
  ) {
    return _list_flags(
      engine_ptr,
    );
  }

  late final _list_flagsPtr = _lookup<
      ffi.NativeFunction<
          ffi.Pointer<ffi.Char> Function(ffi.Pointer<ffi.Void>)>>('list_flags');
  late final _list_flags = _list_flagsPtr
      .asFunction<ffi.Pointer<ffi.Char> Function(ffi.Pointer<ffi.Void>)>();

  /// # Safety
  ///
  /// This function will free the memory occupied by the engine.
  void destroy_engine(
    ffi.Pointer<ffi.Void> engine_ptr,
  ) {
    return _destroy_engine(
      engine_ptr,
    );
  }

  late final _destroy_enginePtr =
      _lookup<ffi.NativeFunction<ffi.Void Function(ffi.Pointer<ffi.Void>)>>(
          'destroy_engine');
  late final _destroy_engine =
      _destroy_enginePtr.asFunction<void Function(ffi.Pointer<ffi.Void>)>();

  /// # Safety
  ///
  /// This function will take in a pointer to the string and free the memory.
  /// See Rust the safety section in CString::from_raw.
  void destroy_string(
    ffi.Pointer<ffi.Char> ptr,
  ) {
    return _destroy_string(
      ptr,
    );
  }

  late final _destroy_stringPtr =
      _lookup<ffi.NativeFunction<ffi.Void Function(ffi.Pointer<ffi.Char>)>>(
          'destroy_string');
  late final _destroy_string =
      _destroy_stringPtr.asFunction<void Function(ffi.Pointer<ffi.Char>)>();

  ffi.Pointer<ffi.Void> initialize_engine_ffi(
    ffi.Pointer<ffi.Char> namespace,
    ffi.Pointer<ffi.Char> options,
  ) {
    return _initialize_engine_ffi(
      namespace,
      options,
    );
  }

  late final _initialize_engine_ffiPtr = _lookup<
      ffi.NativeFunction<
          ffi.Pointer<ffi.Void> Function(ffi.Pointer<ffi.Char>,
              ffi.Pointer<ffi.Char>)>>('initialize_engine_ffi');
  late final _initialize_engine_ffi = _initialize_engine_ffiPtr.asFunction<
      ffi.Pointer<ffi.Void> Function(
          ffi.Pointer<ffi.Char>, ffi.Pointer<ffi.Char>)>();

  ffi.Pointer<ffi.Char> evaluate_boolean_ffi(
    ffi.Pointer<ffi.Void> engine,
    ffi.Pointer<ffi.Char> request,
  ) {
    return _evaluate_boolean_ffi(
      engine,
      request,
    );
  }

  late final _evaluate_boolean_ffiPtr = _lookup<
      ffi.NativeFunction<
          ffi.Pointer<ffi.Char> Function(ffi.Pointer<ffi.Void>,
              ffi.Pointer<ffi.Char>)>>('evaluate_boolean_ffi');
  late final _evaluate_boolean_ffi = _evaluate_boolean_ffiPtr.asFunction<
      ffi.Pointer<ffi.Char> Function(
          ffi.Pointer<ffi.Void>, ffi.Pointer<ffi.Char>)>();

  ffi.Pointer<ffi.Char> evaluate_variant_ffi(
    ffi.Pointer<ffi.Void> engine,
    ffi.Pointer<ffi.Char> request,
  ) {
    return _evaluate_variant_ffi(
      engine,
      request,
    );
  }

  late final _evaluate_variant_ffiPtr = _lookup<
      ffi.NativeFunction<
          ffi.Pointer<ffi.Char> Function(ffi.Pointer<ffi.Void>,
              ffi.Pointer<ffi.Char>)>>('evaluate_variant_ffi');
  late final _evaluate_variant_ffi = _evaluate_variant_ffiPtr.asFunction<
      ffi.Pointer<ffi.Char> Function(
          ffi.Pointer<ffi.Void>, ffi.Pointer<ffi.Char>)>();

  ffi.Pointer<ffi.Char> evaluate_batch_ffi(
    ffi.Pointer<ffi.Void> engine,
    ffi.Pointer<ffi.Char> request,
  ) {
    return _evaluate_batch_ffi(
      engine,
      request,
    );
  }

  late final _evaluate_batch_ffiPtr = _lookup<
      ffi.NativeFunction<
          ffi.Pointer<ffi.Char> Function(ffi.Pointer<ffi.Void>,
              ffi.Pointer<ffi.Char>)>>('evaluate_batch_ffi');
  late final _evaluate_batch_ffi = _evaluate_batch_ffiPtr.asFunction<
      ffi.Pointer<ffi.Char> Function(
          ffi.Pointer<ffi.Void>, ffi.Pointer<ffi.Char>)>();

  ffi.Pointer<ffi.Char> list_flags_ffi(
    ffi.Pointer<ffi.Void> engine,
  ) {
    return _list_flags_ffi(
      engine,
    );
  }

  late final _list_flags_ffiPtr = _lookup<
      ffi.NativeFunction<
          ffi.Pointer<ffi.Char> Function(
              ffi.Pointer<ffi.Void>)>>('list_flags_ffi');
  late final _list_flags_ffi = _list_flags_ffiPtr
      .asFunction<ffi.Pointer<ffi.Char> Function(ffi.Pointer<ffi.Void>)>();

  void destroy_engine_ffi(
    ffi.Pointer<ffi.Void> engine,
  ) {
    return _destroy_engine_ffi(
      engine,
    );
  }

  late final _destroy_engine_ffiPtr =
      _lookup<ffi.NativeFunction<ffi.Void Function(ffi.Pointer<ffi.Void>)>>(
          'destroy_engine_ffi');
  late final _destroy_engine_ffi =
      _destroy_engine_ffiPtr.asFunction<void Function(ffi.Pointer<ffi.Void>)>();

  void destroy_string_ffi(
    ffi.Pointer<ffi.Char> str,
  ) {
    return _destroy_string_ffi(
      str,
    );
  }

  late final _destroy_string_ffiPtr =
      _lookup<ffi.NativeFunction<ffi.Void Function(ffi.Pointer<ffi.Char>)>>(
          'destroy_string_ffi');
  late final _destroy_string_ffi =
      _destroy_string_ffiPtr.asFunction<void Function(ffi.Pointer<ffi.Char>)>();
}
