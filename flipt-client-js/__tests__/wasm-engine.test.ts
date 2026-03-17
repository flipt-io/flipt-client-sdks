// @ts-nocheck
import { jest, describe, it, expect } from '@jest/globals';
import { WasmEngine } from '../src/core/engines/wasm-engine';

describe('WasmEngine', () => {
  const mockNamespace = 'default';

  function createMockWasmEngine(overrides: Record<string, any> = {}) {
    return {
      evaluate_variant: jest.fn().mockReturnValue({
        status: 'success',
        result: {
          flag_key: 'flag1',
          match: true,
          reason: 'MATCH_EVALUATION_REASON',
          segment_keys: ['segment1'],
          variant_key: 'variant1',
          variant_attachment: null,
          request_duration_millis: 1.5,
          timestamp: '2024-01-01T00:00:00Z'
        },
        errorMessage: ''
      }),
      evaluate_boolean: jest.fn().mockReturnValue({
        status: 'success',
        result: {
          flag_key: 'flag_boolean',
          enabled: true,
          reason: 'MATCH_EVALUATION_REASON',
          request_duration_millis: 0.8,
          timestamp: '2024-01-01T00:00:00Z',
          segment_keys: ['segment1']
        },
        errorMessage: ''
      }),
      evaluate_batch: jest.fn().mockReturnValue({
        status: 'success',
        result: {
          responses: [
            {
              type: 'VARIANT_EVALUATION_RESPONSE_TYPE',
              variant_evaluation_response: {
                flag_key: 'flag1',
                match: true,
                reason: 'MATCH_EVALUATION_REASON',
                segment_keys: ['segment1'],
                variant_key: 'variant1',
                variant_attachment: null,
                request_duration_millis: 1.0,
                timestamp: '2024-01-01T00:00:00Z'
              },
              boolean_evaluation_response: null,
              error_evaluation_response: null
            },
            {
              type: 'BOOLEAN_EVALUATION_RESPONSE_TYPE',
              boolean_evaluation_response: {
                flag_key: 'flag_boolean',
                enabled: true,
                reason: 'MATCH_EVALUATION_REASON',
                request_duration_millis: 0.5,
                timestamp: '2024-01-01T00:00:00Z',
                segment_keys: ['segment1']
              },
              variant_evaluation_response: null,
              error_evaluation_response: null
            },
            {
              type: 'ERROR_EVALUATION_RESPONSE_TYPE',
              error_evaluation_response: {
                flag_key: 'notfound',
                namespace_key: 'default',
                reason: 'NOT_FOUND_ERROR_EVALUATION_REASON'
              },
              variant_evaluation_response: null,
              boolean_evaluation_response: null
            }
          ],
          request_duration_millis: 3.2
        },
        errorMessage: ''
      }),
      list_flags: jest.fn().mockReturnValue({
        status: 'success',
        result: [
          {
            key: 'flag1',
            name: 'Flag 1',
            type: 'VARIANT_FLAG_TYPE',
            enabled: true
          },
          {
            key: 'flag_boolean',
            name: 'Boolean Flag',
            type: 'BOOLEAN_FLAG_TYPE',
            enabled: true
          }
        ],
        errorMessage: ''
      }),
      snapshot: jest.fn(),
      ...overrides
    };
  }

  describe('evaluateVariant', () => {
    it('passes a plain object (not JSON string) to the WASM engine', () => {
      const mockEngine = createMockWasmEngine();
      const engine = new WasmEngine(mockEngine, mockNamespace);

      engine.evaluateVariant({
        flagKey: 'flag1',
        entityId: 'user1',
        context: { fizz: 'buzz' }
      });

      expect(mockEngine.evaluate_variant).toHaveBeenCalledWith({
        flag_key: 'flag1',
        entity_id: 'user1',
        context: { fizz: 'buzz' }
      });

      // Must NOT be called with a JSON string
      const arg = mockEngine.evaluate_variant.mock.calls[0][0];
      expect(typeof arg).toBe('object');
      expect(typeof arg).not.toBe('string');
    });

    it('deserializes snake_case response to camelCase', () => {
      const mockEngine = createMockWasmEngine();
      const engine = new WasmEngine(mockEngine, mockNamespace);

      const result = engine.evaluateVariant({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      expect(result.flagKey).toBe('flag1');
      expect(result.match).toBe(true);
      expect(result.reason).toBe('MATCH_EVALUATION_REASON');
      expect(result.segmentKeys).toEqual(['segment1']);
      expect(result.variantKey).toBe('variant1');
      expect(result.requestDurationMillis).toBe(1.5);
    });

    it('throws on null result', () => {
      const mockEngine = createMockWasmEngine({
        evaluate_variant: jest.fn().mockReturnValue(null)
      });
      const engine = new WasmEngine(mockEngine, mockNamespace);

      expect(() =>
        engine.evaluateVariant({
          flagKey: 'flag1',
          entityId: 'user1',
          context: {}
        })
      ).toThrow('Failed to evaluate variant');
    });

    it('throws on failure status', () => {
      const mockEngine = createMockWasmEngine({
        evaluate_variant: jest.fn().mockReturnValue({
          status: 'failure',
          result: null,
          errorMessage: 'flag not found'
        })
      });
      const engine = new WasmEngine(mockEngine, mockNamespace);

      expect(() =>
        engine.evaluateVariant({
          flagKey: 'nonexistent',
          entityId: 'user1',
          context: {}
        })
      ).toThrow('flag not found');
    });
  });

  describe('evaluateBoolean', () => {
    it('passes a plain object to the WASM engine', () => {
      const mockEngine = createMockWasmEngine();
      const engine = new WasmEngine(mockEngine, mockNamespace);

      engine.evaluateBoolean({
        flagKey: 'flag_boolean',
        entityId: 'user1',
        context: { key: 'value' }
      });

      const arg = mockEngine.evaluate_boolean.mock.calls[0][0];
      expect(typeof arg).toBe('object');
      expect(arg.flag_key).toBe('flag_boolean');
      expect(arg.entity_id).toBe('user1');
    });

    it('deserializes response correctly', () => {
      const mockEngine = createMockWasmEngine();
      const engine = new WasmEngine(mockEngine, mockNamespace);

      const result = engine.evaluateBoolean({
        flagKey: 'flag_boolean',
        entityId: 'user1',
        context: {}
      });

      expect(result.flagKey).toBe('flag_boolean');
      expect(result.enabled).toBe(true);
      expect(result.reason).toBe('MATCH_EVALUATION_REASON');
    });

    it('throws on failure', () => {
      const mockEngine = createMockWasmEngine({
        evaluate_boolean: jest.fn().mockReturnValue({
          status: 'failure',
          result: null,
          errorMessage: 'evaluation error'
        })
      });
      const engine = new WasmEngine(mockEngine, mockNamespace);

      expect(() =>
        engine.evaluateBoolean({
          flagKey: 'bad',
          entityId: 'user1',
          context: {}
        })
      ).toThrow('evaluation error');
    });
  });

  describe('evaluateBatch', () => {
    it('passes array of plain objects (not strings) to WASM engine', () => {
      const mockEngine = createMockWasmEngine();
      const engine = new WasmEngine(mockEngine, mockNamespace);

      engine.evaluateBatch([
        { flagKey: 'flag1', entityId: 'user1', context: {} },
        { flagKey: 'flag_boolean', entityId: 'user1', context: {} }
      ]);

      const arg = mockEngine.evaluate_batch.mock.calls[0][0];
      expect(Array.isArray(arg)).toBe(true);
      expect(typeof arg[0]).toBe('object');
      expect(arg[0].flag_key).toBe('flag1');
      expect(arg[1].flag_key).toBe('flag_boolean');
    });

    it('deserializes batch response with correct types', () => {
      const mockEngine = createMockWasmEngine();
      const engine = new WasmEngine(mockEngine, mockNamespace);

      const result = engine.evaluateBatch([
        { flagKey: 'flag1', entityId: 'user1', context: {} },
        { flagKey: 'flag_boolean', entityId: 'user1', context: {} },
        { flagKey: 'notfound', entityId: 'user1', context: {} }
      ]);

      expect(result.responses).toHaveLength(3);

      // Variant response
      const variant = result.responses[0];
      expect(variant.type).toBe('VARIANT_EVALUATION_RESPONSE_TYPE');
      expect(variant.variantEvaluationResponse.flagKey).toBe('flag1');
      expect(variant.variantEvaluationResponse.match).toBe(true);

      // Boolean response
      const boolean = result.responses[1];
      expect(boolean.type).toBe('BOOLEAN_EVALUATION_RESPONSE_TYPE');
      expect(boolean.booleanEvaluationResponse.flagKey).toBe('flag_boolean');
      expect(boolean.booleanEvaluationResponse.enabled).toBe(true);

      // Error response
      const error = result.responses[2];
      expect(error.type).toBe('ERROR_EVALUATION_RESPONSE_TYPE');
      expect(error.errorEvaluationResponse.flagKey).toBe('notfound');
    });

    it('reads request_duration_millis in snake_case from WASM', () => {
      const mockEngine = createMockWasmEngine();
      const engine = new WasmEngine(mockEngine, mockNamespace);

      const result = engine.evaluateBatch([
        { flagKey: 'flag1', entityId: 'user1', context: {} }
      ]);

      expect(result.requestDurationMillis).toBe(3.2);
    });
  });

  describe('listFlags', () => {
    it('deserializes flags to camelCase', () => {
      const mockEngine = createMockWasmEngine();
      const engine = new WasmEngine(mockEngine, mockNamespace);

      const flags = engine.listFlags();
      expect(flags).toHaveLength(2);
      expect(flags[0].key).toBe('flag1');
      expect(flags[1].key).toBe('flag_boolean');
    });

    it('throws on failure', () => {
      const mockEngine = createMockWasmEngine({
        list_flags: jest.fn().mockReturnValue({
          status: 'failure',
          result: null,
          errorMessage: 'failed to list'
        })
      });
      const engine = new WasmEngine(mockEngine, mockNamespace);

      expect(() => engine.listFlags()).toThrow('failed to list');
    });
  });

  describe('snapshot', () => {
    it('delegates to WASM engine', () => {
      const mockEngine = createMockWasmEngine();
      const engine = new WasmEngine(mockEngine, mockNamespace);

      const data = { namespace: { key: 'default' }, flags: [] };
      engine.snapshot(data);

      expect(mockEngine.snapshot).toHaveBeenCalledWith(data);
    });
  });

  describe('isAsync', () => {
    it('is not set (synchronous engine)', () => {
      const mockEngine = createMockWasmEngine();
      const engine = new WasmEngine(mockEngine, mockNamespace);
      expect((engine as any).isAsync).toBeUndefined();
    });
  });
});
