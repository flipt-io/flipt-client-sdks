// @ts-nocheck
import { jest, describe, it, expect, beforeEach } from '@jest/globals';
import { RestEvaluationEngine } from '../src/core/engines/rest-engine';

function createMockClient() {
  return {
    evaluateVariant: jest.fn(),
    evaluateBoolean: jest.fn(),
    evaluateBatch: jest.fn(),
    listFlags: jest.fn()
  };
}

describe('RestEvaluationEngine', () => {
  function createEngine() {
    const engine = new RestEvaluationEngine(
      'http://localhost:8080',
      'default',
      { Authorization: 'Bearer test' }
    );
    const mockClient = createMockClient();
    // Inject mock client (bypassing private access)
    (engine as any).client = mockClient;
    return { engine, mockClient };
  }

  describe('isAsync', () => {
    it('is true for REST engine', () => {
      const { engine } = createEngine();
      expect(engine.isAsync).toBe(true);
    });
  });

  describe('evaluateVariant', () => {
    it('returns a Promise', () => {
      const { engine, mockClient } = createEngine();
      mockClient.evaluateVariant.mockResolvedValue({
        flag_key: 'flag1',
        match: true,
        reason: 'MATCH_EVALUATION_REASON',
        segment_keys: ['seg1'],
        variant_key: 'variant1'
      });

      const result = engine.evaluateVariant({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      expect(result).toBeInstanceOf(Promise);
    });

    it('transforms snake_case response to camelCase', async () => {
      const { engine, mockClient } = createEngine();
      mockClient.evaluateVariant.mockResolvedValue({
        flag_key: 'flag1',
        match: true,
        reason: 'MATCH_EVALUATION_REASON',
        segment_keys: ['segment1'],
        variant_key: 'variant1',
        variant_attachment: null,
        request_duration_millis: 2.3,
        timestamp: '2024-01-01T00:00:00Z'
      });

      const result = await engine.evaluateVariant({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      expect(result.flagKey).toBe('flag1');
      expect(result.match).toBe(true);
      expect(result.reason).toBe('MATCH_EVALUATION_REASON');
      expect(result.segmentKeys).toEqual(['segment1']);
      expect(result.variantKey).toBe('variant1');
      expect(result.requestDurationMillis).toBe(2.3);
    });

    it('delegates to RestClient', async () => {
      const { engine, mockClient } = createEngine();
      mockClient.evaluateVariant.mockResolvedValue({
        flag_key: 'flag1',
        match: false
      });

      await engine.evaluateVariant({
        flagKey: 'flag1',
        entityId: 'user1',
        context: { key: 'value' }
      });

      expect(mockClient.evaluateVariant).toHaveBeenCalledWith({
        flagKey: 'flag1',
        entityId: 'user1',
        context: { key: 'value' }
      });
    });
  });

  describe('evaluateBoolean', () => {
    it('transforms response correctly', async () => {
      const { engine, mockClient } = createEngine();
      mockClient.evaluateBoolean.mockResolvedValue({
        flag_key: 'flag_boolean',
        enabled: true,
        reason: 'MATCH_EVALUATION_REASON',
        request_duration_millis: 1.0,
        segment_keys: ['seg1']
      });

      const result = await engine.evaluateBoolean({
        flagKey: 'flag_boolean',
        entityId: 'user1',
        context: {}
      });

      expect(result.flagKey).toBe('flag_boolean');
      expect(result.enabled).toBe(true);
      expect(result.reason).toBe('MATCH_EVALUATION_REASON');
    });
  });

  describe('evaluateBatch', () => {
    it('transforms batch response with all types', async () => {
      const { engine, mockClient } = createEngine();
      mockClient.evaluateBatch.mockResolvedValue({
        responses: [
          {
            type: 'VARIANT_EVALUATION_RESPONSE_TYPE',
            variant_evaluation_response: {
              flag_key: 'flag1',
              match: true,
              reason: 'MATCH_EVALUATION_REASON',
              segment_keys: ['segment1'],
              variant_key: 'variant1'
            }
          },
          {
            type: 'BOOLEAN_EVALUATION_RESPONSE_TYPE',
            boolean_evaluation_response: {
              flag_key: 'flag_boolean',
              enabled: true,
              reason: 'MATCH_EVALUATION_REASON'
            }
          },
          {
            type: 'ERROR_EVALUATION_RESPONSE_TYPE',
            error_evaluation_response: {
              flag_key: 'notfound',
              namespace_key: 'default',
              reason: 'NOT_FOUND_ERROR_EVALUATION_REASON'
            }
          }
        ],
        request_duration_millis: 5.0
      });

      const result = await engine.evaluateBatch([
        { flagKey: 'flag1', entityId: 'user1', context: {} },
        { flagKey: 'flag_boolean', entityId: 'user1', context: {} },
        { flagKey: 'notfound', entityId: 'user1', context: {} }
      ]);

      expect(result.responses).toHaveLength(3);

      // Variant
      expect(result.responses[0].type).toBe(
        'VARIANT_EVALUATION_RESPONSE_TYPE'
      );
      expect(result.responses[0].variantEvaluationResponse.flagKey).toBe(
        'flag1'
      );

      // Boolean
      expect(result.responses[1].type).toBe(
        'BOOLEAN_EVALUATION_RESPONSE_TYPE'
      );
      expect(result.responses[1].booleanEvaluationResponse.flagKey).toBe(
        'flag_boolean'
      );

      // Error
      expect(result.responses[2].type).toBe('ERROR_EVALUATION_RESPONSE_TYPE');
      expect(result.responses[2].errorEvaluationResponse.flag_key).toBe(
        'notfound'
      );

      expect(result.requestDurationMillis).toBe(5.0);
    });

    it('filters null responses', async () => {
      const { engine, mockClient } = createEngine();
      mockClient.evaluateBatch.mockResolvedValue({
        responses: [
          { unknown_field: {} } // Unknown type → should be filtered
        ],
        request_duration_millis: 0
      });

      const result = await engine.evaluateBatch([
        { flagKey: 'flag1', entityId: 'user1', context: {} }
      ]);

      expect(result.responses).toHaveLength(0);
    });
  });

  describe('listFlags', () => {
    it('returns array from flags property', async () => {
      const { engine, mockClient } = createEngine();
      mockClient.listFlags.mockResolvedValue({
        flags: [
          { key: 'flag1', name: 'Flag 1' },
          { key: 'flag2', name: 'Flag 2' }
        ]
      });

      const result = await engine.listFlags();
      expect(result).toHaveLength(2);
      expect(result[0].key).toBe('flag1');
    });

    it('returns array directly if response is array', async () => {
      const { engine, mockClient } = createEngine();
      mockClient.listFlags.mockResolvedValue([
        { key: 'flag1' },
        { key: 'flag2' }
      ]);

      const result = await engine.listFlags();
      expect(result).toHaveLength(2);
    });

    it('returns empty array on unexpected format', async () => {
      const { engine, mockClient } = createEngine();
      mockClient.listFlags.mockResolvedValue({ something: 'else' });

      const result = await engine.listFlags();
      expect(result).toEqual([]);
    });
  });

  describe('snapshot', () => {
    it('is a no-op', () => {
      const { engine } = createEngine();
      // Should not throw
      engine.snapshot({ any: 'data' });
    });
  });
});
