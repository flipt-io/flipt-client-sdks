import 'package:flipt_client/flipt_client.dart';

/// Starts a client with a listener, closes it, and returns. Used by
/// state_update_listener_test.dart to check the process exits.
void main(List<String> args) {
  final client = FliptClient(
    options: Options(
      url: args[0],
      errorStrategy: ErrorStrategy.fallback,
      onStateUpdate: () {},
    ),
  );
  client.close();
  print('closed');
}
