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
      final EvaluationResponse<VariantResult> result =
          await client.evaluateVariant(
        flagKey: 'flag1',
        entityId: 'someentity',
        context: {'fizz': 'buzz'},
      );

      expect(result.status, equals(EvaluationStatus.success));
      expect(result.result, isA<VariantResult>());

      final variantResult = result.result as VariantResult;
      expect(variantResult.flagKey, equals('flag1'));
      expect(variantResult.reason, equals('MATCH_EVALUATION_REASON'));
      expect(variantResult.segmentKeys, equals(['segment1']));
    });

    test('Evaluate Boolean Flag', () async {
      final EvaluationResponse<BooleanResult> result =
          await client.evaluateBoolean(
        flagKey: 'flag_boolean',
        entityId: 'someentity',
        context: {'fizz': 'buzz'},
      );

      expect(result.status, equals(EvaluationStatus.success));
      expect(result.result, isA<BooleanResult>());

      final booleanResult = result.result as BooleanResult;
      expect(booleanResult.enabled, isA<bool>());
      expect(booleanResult.flagKey, equals('flag_boolean'));
      expect(booleanResult.reason, equals('MATCH_EVALUATION_REASON'));
    });
  });
}
