// @ts-nocheck
import { jest, describe, it, expect } from '@jest/globals';
import { BaseFliptClient } from '../src/core/base';
import { IEvaluationEngine } from '../src/core/engines/types';

// Create a concrete test class since BaseFliptClient is abstract
class TestFliptClient extends BaseFliptClient {
  constructor(engine: IEvaluationEngine, fetcher: any) {
    super(engine, fetcher);
  }

  // Expose protected fields for testing
  setHook(hook: any) {
    this.hook = hook;
  }

  setErrorStrategy(strategy: any) {
    this.errorStrategy = strategy;
  }
}

function createSyncEngine(overrides: Partial<IEvaluationEngine> = {}): IEvaluationEngine {
  return {
    evaluateVariant: jest.fn().mockReturnValue({
      flagKey: 'flag1',
      match: true,
      reason: 'MATCH_EVALUATION_REASON',
      segmentKeys: ['segment1'],
      variantKey: 'variant1',
      requestDurationMillis: 1.0
    }),
    evaluateBoolean: jest.fn().mockReturnValue({
      flagKey: 'flag_boolean',
      enabled: true,
      reason: 'MATCH_EVALUATION_REASON',
      segmentKeys: ['segment1'],
      requestDurationMillis: 0.5
    }),
    evaluateBatch: jest.fn().mockReturnValue({
      responses: [
        {
          type: 'VARIANT_EVALUATION_RESPONSE_TYPE',
          variantEvaluationResponse: {
            flagKey: 'flag1',
            match: true,
            reason: 'MATCH_EVALUATION_REASON',
            segmentKeys: ['segment1'],
            variantKey: 'variant1'
          }
        },
        {
          type: 'BOOLEAN_EVALUATION_RESPONSE_TYPE',
          booleanEvaluationResponse: {
            flagKey: 'flag_boolean',
            enabled: true,
            reason: 'MATCH_EVALUATION_REASON',
            segmentKeys: ['segment1']
          }
        }
      ],
      requestDurationMillis: 2.0
    }),
    listFlags: jest.fn().mockReturnValue([
      { key: 'flag1', name: 'Flag 1' },
      { key: 'flag_boolean', name: 'Boolean Flag' }
    ]),
    snapshot: jest.fn(),
    ...overrides
  };
}

function createAsyncEngine(overrides: Partial<IEvaluationEngine> = {}): IEvaluationEngine {
  return {
    isAsync: true,
    evaluateVariant: jest.fn().mockResolvedValue({
      flagKey: 'flag1',
      match: true,
      reason: 'MATCH_EVALUATION_REASON',
      segmentKeys: ['segment1'],
      variantKey: 'variant1'
    }),
    evaluateBoolean: jest.fn().mockResolvedValue({
      flagKey: 'flag_boolean',
      enabled: true,
      reason: 'MATCH_EVALUATION_REASON',
      segmentKeys: ['segment1']
    }),
    evaluateBatch: jest.fn().mockResolvedValue({
      responses: [
        {
          type: 'VARIANT_EVALUATION_RESPONSE_TYPE',
          variantEvaluationResponse: {
            flagKey: 'flag1',
            match: true,
            reason: 'MATCH_EVALUATION_REASON',
            segmentKeys: ['segment1'],
            variantKey: 'variant1'
          }
        }
      ],
      requestDurationMillis: 5.0
    }),
    listFlags: jest.fn().mockResolvedValue([{ key: 'flag1' }]),
    snapshot: jest.fn(),
    ...overrides
  };
}

const mockFetcher = jest.fn();

describe('BaseFliptClient', () => {
  describe('sync methods with sync engine', () => {
    it('evaluateVariant returns result synchronously', () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      const result = client.evaluateVariant({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      expect(result.flagKey).toBe('flag1');
      expect(result.match).toBe(true);
      expect(result.variantKey).toBe('variant1');
    });

    it('evaluateBoolean returns result synchronously', () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      const result = client.evaluateBoolean({
        flagKey: 'flag_boolean',
        entityId: 'user1',
        context: {}
      });

      expect(result.enabled).toBe(true);
    });

    it('evaluateBatch returns results synchronously', () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      const result = client.evaluateBatch([
        { flagKey: 'flag1', entityId: 'user1', context: {} },
        { flagKey: 'flag_boolean', entityId: 'user1', context: {} }
      ]);

      expect(result.responses).toHaveLength(2);
    });

    it('listFlags returns flags synchronously', () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      const flags = client.listFlags();
      expect(flags).toHaveLength(2);
    });
  });

  describe('sync methods with async engine (should throw)', () => {
    it('evaluateVariant throws when engine is async', () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      expect(() =>
        client.evaluateVariant({
          flagKey: 'flag1',
          entityId: 'user1',
          context: {}
        })
      ).toThrow(
        'evaluateVariant is synchronous and not available in REST mode. Use evaluateVariantAsync instead.'
      );
    });

    it('evaluateBoolean throws when engine is async', () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      expect(() =>
        client.evaluateBoolean({
          flagKey: 'flag1',
          entityId: 'user1',
          context: {}
        })
      ).toThrow(
        'evaluateBoolean is synchronous and not available in REST mode. Use evaluateBooleanAsync instead.'
      );
    });

    it('evaluateBatch throws when engine is async', () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      expect(() =>
        client.evaluateBatch([
          { flagKey: 'flag1', entityId: 'user1', context: {} }
        ])
      ).toThrow(
        'evaluateBatch is synchronous and not available in REST mode. Use evaluateBatchAsync instead.'
      );
    });

    it('listFlags throws when engine is async', () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      expect(() => client.listFlags()).toThrow(
        'listFlags is synchronous and not available in REST mode. Use listFlagsAsync instead.'
      );
    });
  });

  describe('async methods', () => {
    it('evaluateVariantAsync works with sync engine', async () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      const result = await client.evaluateVariantAsync({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      expect(result.flagKey).toBe('flag1');
      expect(result.match).toBe(true);
    });

    it('evaluateVariantAsync works with async engine', async () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      const result = await client.evaluateVariantAsync({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      expect(result.flagKey).toBe('flag1');
    });

    it('evaluateBooleanAsync works with async engine', async () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      const result = await client.evaluateBooleanAsync({
        flagKey: 'flag_boolean',
        entityId: 'user1',
        context: {}
      });

      expect(result.enabled).toBe(true);
    });

    it('evaluateBatchAsync works with async engine', async () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      const result = await client.evaluateBatchAsync([
        { flagKey: 'flag1', entityId: 'user1', context: {} }
      ]);

      expect(result.responses).toHaveLength(1);
    });

    it('listFlagsAsync works with async engine', async () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      const flags = await client.listFlagsAsync();
      expect(flags).toHaveLength(1);
    });
  });

  describe('input validation', () => {
    it('throws on empty flagKey for evaluateVariant', () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      expect(() =>
        client.evaluateVariant({ flagKey: '', entityId: 'user1', context: {} })
      ).toThrow('flagKey cannot be empty');
    });

    it('throws on empty entityId for evaluateVariant', () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      expect(() =>
        client.evaluateVariant({
          flagKey: 'flag1',
          entityId: '',
          context: {}
        })
      ).toThrow('entityId cannot be empty');
    });

    it('throws on empty flagKey for evaluateBoolean', () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      expect(() =>
        client.evaluateBoolean({ flagKey: '', entityId: 'user1', context: {} })
      ).toThrow('flagKey cannot be empty');
    });

    it('throws on whitespace-only flagKey for async variant', async () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      await expect(
        client.evaluateVariantAsync({
          flagKey: '   ',
          entityId: 'user1',
          context: {}
        })
      ).rejects.toThrow('flagKey cannot be empty');
    });

    it('throws on whitespace-only entityId for async boolean', async () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);

      await expect(
        client.evaluateBooleanAsync({
          flagKey: 'flag1',
          entityId: '   ',
          context: {}
        })
      ).rejects.toThrow('entityId cannot be empty');
    });
  });

  describe('hooks', () => {
    it('calls before and after hooks on evaluateVariant', () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);
      const hook = { before: jest.fn(), after: jest.fn() };
      client.setHook(hook);

      client.evaluateVariant({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      expect(hook.before).toHaveBeenCalledWith({ flagKey: 'flag1' });
      expect(hook.after).toHaveBeenCalledWith(
        expect.objectContaining({
          flagKey: 'flag1',
          flagType: 'variant',
          value: 'variant1'
        })
      );
    });

    it('calls before and after hooks on evaluateBoolean', () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);
      const hook = { before: jest.fn(), after: jest.fn() };
      client.setHook(hook);

      client.evaluateBoolean({
        flagKey: 'flag_boolean',
        entityId: 'user1',
        context: {}
      });

      expect(hook.before).toHaveBeenCalledWith({ flagKey: 'flag_boolean' });
      expect(hook.after).toHaveBeenCalledWith(
        expect.objectContaining({
          flagKey: 'flag_boolean',
          flagType: 'boolean',
          value: 'true'
        })
      );
    });

    it('calls before hooks for each request in batch', () => {
      const engine = createSyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);
      const hook = { before: jest.fn(), after: jest.fn() };
      client.setHook(hook);

      client.evaluateBatch([
        { flagKey: 'flag1', entityId: 'user1', context: {} },
        { flagKey: 'flag_boolean', entityId: 'user1', context: {} }
      ]);

      expect(hook.before).toHaveBeenCalledTimes(2);
      expect(hook.before).toHaveBeenCalledWith({ flagKey: 'flag1' });
      expect(hook.before).toHaveBeenCalledWith({ flagKey: 'flag_boolean' });
    });

    it('calls hooks on async variant', async () => {
      const engine = createAsyncEngine();
      const client = new TestFliptClient(engine, mockFetcher);
      const hook = { before: jest.fn(), after: jest.fn() };
      client.setHook(hook);

      await client.evaluateVariantAsync({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      expect(hook.before).toHaveBeenCalledWith({ flagKey: 'flag1' });
      expect(hook.after).toHaveBeenCalledWith(
        expect.objectContaining({
          flagKey: 'flag1',
          flagType: 'variant'
        })
      );
    });
  });

  describe('refresh', () => {
    it('calls fetcher and updates engine snapshot', async () => {
      const engine = createSyncEngine();
      const newData = { namespace: { key: 'default' }, flags: [] };
      const fetcher = jest.fn().mockResolvedValue({
        ok: true,
        headers: {
          get: (key: string) =>
            key === 'etag' ? '"new-etag"' : null
        },
        json: async () => newData
      });

      const client = new TestFliptClient(engine, fetcher);

      const updated = await client.refresh();
      expect(updated).toBe(true);
      expect(engine.snapshot).toHaveBeenCalledWith(newData);
    });

    it('returns false when etag matches', async () => {
      const engine = createSyncEngine();
      const fetcher = jest.fn().mockResolvedValue({
        ok: true,
        headers: {
          get: (key: string) =>
            key === 'etag' ? '"same-etag"' : null
        },
        json: async () => ({})
      });

      const client = new TestFliptClient(engine, fetcher);

      // First call sets the etag
      await client.refresh();

      // Second call with same etag
      const updated = await client.refresh();
      expect(updated).toBe(false);
    });
  });
});
