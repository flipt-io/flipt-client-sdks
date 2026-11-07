import 'dart:async';
import 'dart:convert';
import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';

import 'package:ffi/ffi.dart';

import 'errors.dart';
import 'ffi/bindings.dart';

/// A registered state update listener. Closing it unsubscribes; closing twice
/// is a no-op.
class StateUpdateSubscription {
  void Function()? _unsubscribe;

  StateUpdateSubscription._(this._unsubscribe);

  /// Removes the listener.
  void close() {
    final unsubscribe = _unsubscribe;
    _unsubscribe = null;
    unsubscribe?.call();
  }
}

const _changed = 'changed';

typedef _WaitNative = Pointer<Char> Function(Pointer<Void>);
typedef _DestroyStringNative = Void Function(Pointer<Char>);
typedef _DestroyString = void Function(Pointer<Char>);

/// Owns the helper isolate that waits on the engine for flag state changes,
/// and fans out to the client's listeners on the client's isolate. Internal to
/// FliptClient.
///
/// An FFI call blocks its whole isolate, so the blocking wait runs on a helper
/// isolate. It sends one message per change and then waits for an ack before
/// waiting again, so changes that land while listeners run are coalesced by
/// the engine into a single follow-up call.
class StateUpdates {
  final FliptEngine _bindings;
  final DynamicLibrary _lib;
  final Pointer<Void> _engine;
  final List<void Function()> _listeners = [];
  bool _closed = false;

  // Set while a helper isolate is running. The helper never frees the
  // subscription: this isolate does, after the helper has exited, so nobody
  // can be waiting on it.
  Pointer<Void>? _subscription;
  ReceivePort? _port;
  SendPort? _ack;

  StateUpdates(this._lib, this._bindings, this._engine);

  StateUpdateSubscription add(void Function() listener) {
    if (_closed) {
      throw FliptError('client is closed');
    }
    _ensureIsolate();
    // A distinct wrapper per call, so adding the same function twice yields
    // two independent subscriptions.
    void entry() => listener();
    _listeners.add(entry);
    return StateUpdateSubscription._(() => _listeners.remove(entry));
  }

  /// Stops notifications. Listeners run on this isolate, so none can be
  /// running on another thread when this returns, and none is called after.
  /// Returns false if already closed.
  bool close() {
    if (_closed) {
      return false;
    }
    _closed = true;
    _listeners.clear();
    final subscription = _subscription;
    if (subscription != null) {
      // Wakes the helper if it is waiting. The subscription holds its own
      // reference to the engine's update state, so the helper can finish
      // after the engine is destroyed.
      // A helper waiting for an ack still gets one: its "changed" message is
      // queued on the port, and the handler acks it. Its next wait returns
      // "closed".
      _bindings.close_state_subscription(subscription);
    }
    return true;
  }

  void _ensureIsolate() {
    if (_subscription != null) {
      return;
    }
    final subscription = _bindings.subscribe_state_updates(_engine);
    if (subscription == nullptr) {
      throw EvaluationError('Failed to subscribe to state updates');
    }
    final port = ReceivePort();
    _subscription = subscription;
    _port = port;
    port.listen((message) => _onMessage(port, subscription, message));
    // The helper calls these by address rather than reloading the library,
    // which could resolve differently (desktop paths depend on the working
    // directory).
    final wait =
        _lib.lookup<NativeFunction<_WaitNative>>('wait_for_state_update');
    final destroyString =
        _lib.lookup<NativeFunction<_DestroyStringNative>>('destroy_string');
    Isolate.spawn(
      _waitLoop,
      [
        port.sendPort,
        subscription.address,
        wait.address,
        destroyString.address,
      ],
      debugName: 'flipt-state-update',
      onExit: port.sendPort,
      onError: port.sendPort,
    ).catchError((Object e) {
      stderr.writeln('[flipt] Failed to start state update isolate: $e');
      _onExit(port, subscription);
      return Isolate.current;
    });
  }

  void _onMessage(
      ReceivePort port, Pointer<Void> subscription, Object? message) {
    if (message is SendPort) {
      _ack = message;
    } else if (message == _changed) {
      _dispatch();
      _ack?.send(null);
    } else if (message is List) {
      // An uncaught error from the helper, as [error, stack].
      stderr.writeln('[flipt] State update isolate failed: ${message.first}');
    } else if (message is String) {
      stderr.writeln('[flipt] Failed to wait for state update: $message');
    } else if (message == null) {
      _onExit(port, subscription);
    }
  }

  void _dispatch() {
    // A copy, so listeners may add or remove listeners while this runs.
    for (final listener in List.of(_listeners)) {
      if (_closed) {
        return;
      }
      try {
        listener();
      } catch (e, stack) {
        stderr.writeln('[flipt] State update listener threw: $e\n$stack');
      }
    }
  }

  void _onExit(ReceivePort port, Pointer<Void> subscription) {
    port.close();
    _bindings.destroy_state_subscription(subscription);
    if (identical(_port, port)) {
      // If the helper died early, the next listener starts a new one.
      _subscription = null;
      _port = null;
      _ack = null;
    }
  }
}

/// Runs on the helper isolate: [SendPort to the client, subscription address,
/// wait_for_state_update address, destroy_string address].
Future<void> _waitLoop(List<Object> args) async {
  final client = args[0] as SendPort;
  final subscription = Pointer<Void>.fromAddress(args[1] as int);
  final waitForStateUpdate =
      Pointer<NativeFunction<_WaitNative>>.fromAddress(args[2] as int)
          .asFunction<_WaitNative>();
  final destroyString =
      Pointer<NativeFunction<_DestroyStringNative>>.fromAddress(args[3] as int)
          .asFunction<_DestroyString>();
  final acks = ReceivePort();
  final ackIterator = StreamIterator(acks);
  client.send(acks.sendPort);
  try {
    while (true) {
      final resultPtr = waitForStateUpdate(subscription);
      if (resultPtr == nullptr) {
        client.send('empty response');
        return;
      }
      final json = resultPtr.cast<Utf8>().toDartString();
      destroyString(resultPtr);
      final result = jsonDecode(json) as Map<String, dynamic>;
      if (result['status'] != 'success') {
        client.send(result['error_message'] as String? ?? 'Unknown error');
        return;
      }
      if (result['result'] != _changed) {
        return;
      }
      client.send(_changed);
      if (!await ackIterator.moveNext()) {
        return;
      }
    }
  } finally {
    acks.close();
  }
}
