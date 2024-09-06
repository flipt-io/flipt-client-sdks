import 'package:flipt_client/flipt_client.dart';
import 'package:test/test.dart';
import 'dart:io' show Platform;

void main() {
  group('FliptEvaluationClient Tests', () {
    late FliptEvaluationClient client;

    setUp(() {
      Map<String, String> envVars = Platform.environment;

      String url = envVars['FLIPT_URL'] ?? 'http://localhost:8080';
      String token = envVars['FLIPT_AUTH_TOKEN'] ?? 'secret';

      client = FliptEvaluationClient(
        options: Options.withClientToken(token, url: url),
      );
    });

    tearDown(() {
      client.close();
    });

    test('Evaluate Variant Flag', () async {
      final result = await client.evaluateVariant(
        flagKey: 'flag1',
        entityId: 'someentity',
        context: {'fizz': 'buzz'},
      );

      expect(result.status, equals(Status.success));
      expect(result.result, isA<VariantEvaluationResponse>());

      final variantResult = result.result as VariantEvaluationResponse;
      expect(variantResult.flagKey, equals('flag1'));
      expect(variantResult.reason, equals('MATCH_EVALUATION_REASON'));
      expect(variantResult.segmentKeys, equals(['segment1']));
    });

    test('Evaluate Boolean Flag', () async {
      final result = await client.evaluateBoolean(
        flagKey: 'flag_boolean',
        entityId: 'someentity',
        context: {'fizz': 'buzz'},
      );

      expect(result.status, equals(Status.success));
      expect(result.result, isA<BooleanEvaluationResponse>());

      final booleanResult = result.result as BooleanEvaluationResponse;
      expect(booleanResult.enabled, isA<bool>());
      expect(booleanResult.flagKey, equals('flag_boolean'));
      expect(booleanResult.reason, equals('MATCH_EVALUATION_REASON'));
    });

    test('Evaluate Batch', () async {
      final result = await client.evaluateBatch([
        EvaluationRequest(
          flagKey: 'flag1',
          entityId: 'someentity',
          context: {'fizz': 'buzz'},
        ),
        EvaluationRequest(
          flagKey: 'flag_boolean',
          entityId: 'someentity',
          context: {'fizz': 'buzz'},
        ),
        EvaluationRequest(
          flagKey: 'notfound',
          entityId: 'someentity',
          context: {'fizz': 'buzz'},
        ),
      ]);

      expect(result.status, equals(Status.success));
      expect(result.result, isA<BatchEvaluationResponse>());

      final batch = result.result as BatchEvaluationResponse;

      expect(batch.responses, hasLength(3));

      final variant = batch.responses[0];
      expect(variant.type, equals('VARIANT_EVALUATION_RESPONSE_TYPE'));
      expect(variant.variantEvaluationResponse!.match, isTrue);
      expect(variant.variantEvaluationResponse!.flagKey, equals('flag1'));
      expect(variant.variantEvaluationResponse!.reason,
          equals('MATCH_EVALUATION_REASON'));
      expect(
          variant.variantEvaluationResponse!.segmentKeys, contains('segment1'));

      final boolean = batch.responses[1];
      expect(boolean.type, equals('BOOLEAN_EVALUATION_RESPONSE_TYPE'));
      expect(
          boolean.booleanEvaluationResponse!.flagKey, equals('flag_boolean'));
      expect(boolean.booleanEvaluationResponse!.enabled, isTrue);
      expect(boolean.booleanEvaluationResponse!.reason,
          equals('MATCH_EVALUATION_REASON'));

      final errorResponse = batch.responses[2];
      expect(errorResponse.type, equals('ERROR_EVALUATION_RESPONSE_TYPE'));
      expect(
          errorResponse.errorEvaluationResponse!.flagKey, equals('notfound'));
      expect(errorResponse.errorEvaluationResponse!.namespaceKey,
          equals('default'));
      expect(errorResponse.errorEvaluationResponse!.reason,
          equals('NOT_FOUND_ERROR_EVALUATION_REASON'));
    });

    test('List Flags', () async {
      final result = await client.listFlags();
      expect(result.status, equals(Status.success));
      expect(result.result, isA<List<Flag>>());
      expect(result.result, hasLength(2));
    });
  });
}
