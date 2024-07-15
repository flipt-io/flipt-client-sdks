import 'package:flipt_client_flutter/flipt_client.dart';
import 'package:test/test.dart';

void main() {
  final client = FliptEvaluationClient();

  setUp(() {
    // Additional setup goes here.
  });

  test('Constructor', () {
    expect(client, isNotNull);
  });

  test('Destroy engine', () {
    client.destroy();
  });

  test('Evaluate Variant', () {
    final result = client.evaluateVariant(
        flagKey: 'test', entityId: 'entity_id', context: {'key': 'value'});

    expect(result, isNotNull);
    expect(result.status, 'success');
  });
}
