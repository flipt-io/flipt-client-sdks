import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:isolate';

import 'package:flipt_client/flipt_client.dart';
import 'package:test/test.dart';

const empty = '{"namespace":{"key":"default"},"flags":[]}';
const withFlag =
    '{"namespace":{"key":"default"},"flags":[{"key":"new_flag","name":"New",'
    '"type":"BOOLEAN_FLAG_TYPE","enabled":true}]}';
const barrier =
    '{"namespace":{"key":"default"},"flags":[{"key":"barrier","name":"Barrier",'
    '"type":"BOOLEAN_FLAG_TYPE","enabled":true}]}';
const wait = Duration(seconds: 5);

/// A fake Flipt server on its own isolate. The engine fetches synchronously
/// while the client is created, which blocks the test's isolate, so a server
/// on the same isolate could never answer.
class FakeFlipt {
  final Isolate _isolate;
  final SendPort _commands;
  final int port;

  FakeFlipt._(this._isolate, this._commands, this.port);

  static Future<FakeFlipt> start() async {
    final ready = ReceivePort();
    final isolate = await Isolate.spawn(_serve, ready.sendPort);
    final reply = await ready.first as List;
    return FakeFlipt._(isolate, reply[0] as SendPort, reply[1] as int);
  }

  String get url => 'http://127.0.0.1:$port';

  set body(String body) => _commands.send(['body', body]);

  /// Sends a document on the open stream, or queues it for the next one.
  void stream(String doc) => _commands.send(['stream', doc]);

  void stop() => _isolate.kill(priority: Isolate.immediate);
}

Future<void> _serve(SendPort ready) async {
  var body = empty;
  // Documents the stream endpoint sends, one per line, as the test pushes them.
  final docs = StreamController<String>();
  final commands = ReceivePort();
  commands.listen((message) {
    final command = message as List;
    if (command[0] == 'body') {
      body = command[1] as String;
    } else {
      docs.add(command[1] as String);
    }
  });

  final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
  ready.send([commands.sendPort, server.port]);
  await for (final request in server) {
    final path = request.uri.path;
    final response = request.response;
    response.headers.contentType = ContentType.json;
    if (path.startsWith('/internal/v1/evaluation/snapshot/namespace/default')) {
      response.write(body);
      await response.close();
    } else if (path.startsWith(
        '/client/v2/environments/default/namespaces/default/stream')) {
      response.bufferOutput = false;
      // Only one stream at a time; it stays open until the isolate is killed.
      // Writes wait for the previous flush, which the sink requires.
      unawaited(() async {
        await for (final doc in docs.stream) {
          response.write('{"result":$doc}\n');
          await response.flush();
        }
      }());
    } else {
      response.statusCode = 404;
      await response.close();
    }
  }
}

void main() {
  late FakeFlipt flipt;

  setUp(() async {
    flipt = await FakeFlipt.start();
  });

  tearDown(() {
    flipt.stop();
  });

  FliptClient newClient({
    void Function()? onStateUpdate,
    FetchMode fetchMode = FetchMode.polling,
  }) {
    return FliptClient(
      options: Options(
        url: flipt.url,
        updateInterval: const Duration(seconds: 1),
        errorStrategy: ErrorStrategy.fallback,
        fetchMode: fetchMode,
        onStateUpdate: onStateUpdate,
      ),
    );
  }

  test('fires on change and can evaluate from the listener', () async {
    var calls = 0;
    final evaluated = Completer<bool>();
    late FliptClient client;
    client = newClient(onStateUpdate: () {
      calls++;
      // Evaluating from inside the listener must not deadlock.
      evaluated.complete(client.evaluateBoolean(
          flagKey: 'new_flag', entityId: 'e', context: {}).enabled);
    });
    try {
      // Unchanged polls must not fire the listener.
      await Future<void>.delayed(const Duration(milliseconds: 1500));
      expect(calls, 0);

      flipt.body = withFlag;
      expect(await evaluated.future.timeout(wait), isTrue);
    } finally {
      client.close();
    }
  });

  test('fires on a streamed change', () async {
    final fired = Completer<void>();
    flipt.stream(empty);
    final client = newClient(
      fetchMode: FetchMode.streaming,
      onStateUpdate: () {
        if (!fired.isCompleted) fired.complete();
      },
    );
    try {
      flipt.stream(withFlag);
      await fired.future.timeout(wait);
    } finally {
      client.close();
    }
  });

  test('keeps calling other listeners when one throws', () async {
    final fired = Completer<void>();
    final client = newClient(onStateUpdate: () => throw StateError('boom'));
    try {
      client.addStateUpdateListener(() {
        if (!fired.isCompleted) fired.complete();
      });
      flipt.body = withFlag;
      await fired.future.timeout(wait);
      expect(
          client.evaluateBoolean(
              flagKey: 'new_flag', entityId: 'e', context: {}).enabled,
          isTrue);
    } finally {
      client.close();
    }
  });

  test('can close the client from inside a listener', () async {
    final done = Completer<void>();
    var laterCalls = 0;
    late FliptClient client;
    client = newClient(onStateUpdate: () {
      client.close();
      done.complete();
    });
    try {
      // Registered after the closing listener, so it must never run.
      client.addStateUpdateListener(() => laterCalls++);
      flipt.body = withFlag;
      await done.future.timeout(wait);
      await Future<void>.delayed(const Duration(milliseconds: 300));
      expect(laterCalls, 0);
      expect(
          () =>
              client.evaluateBoolean(flagKey: 'f', entityId: 'e', context: {}),
          throwsA(isA<FliptError>()
              .having((e) => e.message, 'message', 'client is closed')));
    } finally {
      client.close();
    }
  });

  test('adds and removes listeners while running', () async {
    final fired = Completer<void>();
    var removedCalls = 0;
    final client = newClient();
    try {
      final sub = client.addStateUpdateListener(() => removedCalls++);
      sub.close();
      sub.close(); // closing twice is a no-op
      client.addStateUpdateListener(() {
        if (!fired.isCompleted) fired.complete();
      });
      flipt.body = withFlag;
      await fired.future.timeout(wait);
      expect(removedCalls, 0);
    } finally {
      client.close();
    }
  });

  test('can add a listener from inside a listener', () async {
    final added = Completer<void>();
    final addedFired = Completer<void>();
    late FliptClient client;
    client = newClient(onStateUpdate: () {
      if (added.isCompleted) return;
      client.addStateUpdateListener(() {
        if (!addedFired.isCompleted) addedFired.complete();
      });
      added.complete();
    });
    try {
      flipt.body = withFlag;
      await added.future.timeout(wait);
      flipt.body = barrier;
      await addedFired.future.timeout(wait);
    } finally {
      client.close();
    }
  });

  test('throws when adding a listener after close', () {
    // Start the helper isolate first, so the closed check is what throws.
    final client = newClient(onStateUpdate: () {});
    client.close();
    expect(
        () => client.addStateUpdateListener(() {}),
        throwsA(isA<FliptError>()
            .having((e) => e.message, 'message', 'client is closed')));
  });

  test('throws from every method after close', () {
    final client = newClient();
    client.close();
    client.close(); // closing twice is a no-op
    final closed = throwsA(isA<FliptError>()
        .having((e) => e.message, 'message', 'client is closed'));
    expect(
        () => client.evaluateVariant(flagKey: 'f', entityId: 'e', context: {}),
        closed);
    expect(
        () => client.evaluateBatch(
            [EvaluationRequest(flagKey: 'f', entityId: 'e', context: {})]),
        closed);
    expect(() => client.listFlags(), closed);
    expect(() => client.getSnapshot(), closed);
  });

  test('stops calling listeners after close', () async {
    var calls = 0;
    final client = newClient(onStateUpdate: () => calls++);
    client.close();
    flipt.body = withFlag;
    await Future<void>.delayed(const Duration(milliseconds: 1500));
    expect(calls, 0);
  });

  test('a program exits after close', () async {
    // The helper isolate blocks in native code, so close must wake it or the
    // process never exits.
    final result = await Process.run(Platform.resolvedExecutable,
        ['run', 'test/fixtures/close_exits.dart', flipt.url]).timeout(
      const Duration(seconds: 90),
    );
    expect(result.exitCode, 0, reason: '${result.stdout}${result.stderr}');
    expect(result.stdout, contains('closed'));
  }, timeout: const Timeout(Duration(seconds: 120)));

  test('does not send onStateUpdate to the engine', () {
    final options = Options(onStateUpdate: () {});
    final json = jsonEncode(options.toJson());
    expect(json, isNot(contains('state_update')));
    expect(json, isNot(contains('StateUpdate')));
  });
}
