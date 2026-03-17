// @ts-nocheck
import { jest, describe, it, expect, beforeEach, afterAll } from '@jest/globals';
import { RestClient } from '../src/core/engines/rest-client';

// Store original fetch
const originalFetch = globalThis.fetch;

function createMockResponse(body: any, options: { status?: number; ok?: boolean; statusText?: string; headers?: Record<string, string> } = {}) {
  const headers = new Map(Object.entries(options.headers ?? {}));
  return {
    ok: options.ok ?? true,
    status: options.status ?? 200,
    statusText: options.statusText ?? 'OK',
    headers: {
      get: (key: string) => headers.get(key) ?? null
    },
    json: async () => body,
    text: async () => JSON.stringify(body)
  };
}

describe('RestClient', () => {
  let mockFetch: jest.Mock;

  beforeEach(() => {
    mockFetch = jest.fn();
    globalThis.fetch = mockFetch as unknown as typeof fetch;
  });

  afterAll(() => {
    globalThis.fetch = originalFetch;
  });

  describe('URL construction', () => {
    it('strips trailing slash from baseUrl', async () => {
      mockFetch.mockResolvedValue(createMockResponse({ enabled: true }));
      const client = new RestClient('http://localhost:8080/', 'default', {});

      await client.evaluateBoolean({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      const calledUrl = mockFetch.mock.calls[0][0];
      expect(calledUrl).toBe(
        'http://localhost:8080/evaluate/v1/default/boolean'
      );
    });

    it('encodes namespace in URL', async () => {
      mockFetch.mockResolvedValue(createMockResponse({ enabled: true }));
      const client = new RestClient(
        'http://localhost:8080',
        'my namespace/special',
        {}
      );

      await client.evaluateBoolean({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      const calledUrl = mockFetch.mock.calls[0][0];
      expect(calledUrl).toBe(
        'http://localhost:8080/evaluate/v1/my%20namespace%2Fspecial/boolean'
      );
    });
  });

  describe('evaluateVariant', () => {
    it('sends POST with snake_case body', async () => {
      mockFetch.mockResolvedValue(
        createMockResponse({
          flag_key: 'flag1',
          match: true,
          reason: 'MATCH_EVALUATION_REASON',
          segment_keys: ['segment1'],
          variant_key: 'variant1'
        })
      );

      const client = new RestClient('http://localhost:8080', 'default', {
        Authorization: 'Bearer token123'
      });

      await client.evaluateVariant({
        flagKey: 'flag1',
        entityId: 'user1',
        context: { fizz: 'buzz' }
      });

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/evaluate/v1/default/variant',
        expect.objectContaining({
          method: 'POST',
          headers: expect.objectContaining({
            Authorization: 'Bearer token123',
            'Content-Type': 'application/json'
          })
        })
      );

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body.flag_key).toBe('flag1');
      expect(body.entity_id).toBe('user1');
      expect(body.context).toEqual({ fizz: 'buzz' });
      // Should NOT have camelCase keys
      expect(body.flagKey).toBeUndefined();
      expect(body.entityId).toBeUndefined();
    });
  });

  describe('evaluateBoolean', () => {
    it('sends POST to boolean endpoint', async () => {
      mockFetch.mockResolvedValue(
        createMockResponse({ flag_key: 'flag_boolean', enabled: true })
      );

      const client = new RestClient('http://localhost:8080', 'default', {});
      await client.evaluateBoolean({
        flagKey: 'flag_boolean',
        entityId: 'user1',
        context: {}
      });

      const calledUrl = mockFetch.mock.calls[0][0];
      expect(calledUrl).toBe(
        'http://localhost:8080/evaluate/v1/default/boolean'
      );
    });
  });

  describe('evaluateBatch', () => {
    it('sends batch payload with requests array', async () => {
      mockFetch.mockResolvedValue(
        createMockResponse({ responses: [], request_duration_millis: 1.0 })
      );

      const client = new RestClient('http://localhost:8080', 'default', {});
      await client.evaluateBatch([
        { flagKey: 'flag1', entityId: 'user1', context: {} },
        { flagKey: 'flag2', entityId: 'user2', context: { key: 'val' } }
      ]);

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body.requests).toHaveLength(2);
      expect(body.requests[0].flag_key).toBe('flag1');
      expect(body.requests[1].flag_key).toBe('flag2');
    });
  });

  describe('listFlags', () => {
    it('sends GET to flags endpoint', async () => {
      mockFetch.mockResolvedValue(
        createMockResponse({ flags: [{ key: 'flag1' }, { key: 'flag2' }] })
      );

      const client = new RestClient('http://localhost:8080', 'default', {});
      const result = await client.listFlags();

      expect(mockFetch.mock.calls[0][1].method).toBe('GET');
      const calledUrl = mockFetch.mock.calls[0][0];
      expect(calledUrl).toBe(
        'http://localhost:8080/evaluate/v1/default/flags'
      );
    });
  });

  describe('error handling', () => {
    it('throws on non-OK response with server error message', async () => {
      mockFetch.mockResolvedValue(
        createMockResponse(
          { message: 'flag not found' },
          { ok: false, status: 404, statusText: 'Not Found' }
        )
      );

      const client = new RestClient('http://localhost:8080', 'default', {});
      await expect(
        client.evaluateVariant({
          flagKey: 'nonexistent',
          entityId: 'user1',
          context: {}
        })
      ).rejects.toThrow('Flipt evaluation failed: flag not found');
    });

    it('throws on non-OK response with error field', async () => {
      mockFetch.mockResolvedValue(
        createMockResponse(
          { error: 'unauthorized' },
          { ok: false, status: 401, statusText: 'Unauthorized' }
        )
      );

      const client = new RestClient('http://localhost:8080', 'default', {});
      await expect(
        client.evaluateBoolean({
          flagKey: 'flag1',
          entityId: 'user1',
          context: {}
        })
      ).rejects.toThrow('Flipt evaluation failed: unauthorized');
    });

    it('wraps network errors', async () => {
      mockFetch.mockRejectedValue(new Error('fetch failed'));

      const client = new RestClient('http://localhost:8080', 'default', {});
      await expect(
        client.evaluateVariant({
          flagKey: 'flag1',
          entityId: 'user1',
          context: {}
        })
      ).rejects.toThrow('Network error: Unable to connect to Flipt server');
    });

    it('handles 304 Not Modified', async () => {
      mockFetch.mockResolvedValue(
        createMockResponse(null, { ok: false, status: 304, statusText: 'Not Modified' })
      );

      const client = new RestClient('http://localhost:8080', 'default', {});
      const result = await client.evaluateBoolean({
        flagKey: 'flag1',
        entityId: 'user1',
        context: {}
      });

      expect(result._noChange).toBe(true);
    });
  });

  describe('camelToSnake conversion', () => {
    it('converts request keys from camelCase to snake_case', async () => {
      mockFetch.mockResolvedValue(createMockResponse({}));
      const client = new RestClient('http://localhost:8080', 'default', {});

      await client.evaluateVariant({
        flagKey: 'myFlag',
        entityId: 'entity123',
        context: { someKey: 'someValue' }
      });

      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body).toHaveProperty('flag_key', 'myFlag');
      expect(body).toHaveProperty('entity_id', 'entity123');
      // context values are NOT converted (they are user-defined)
      expect(body).toHaveProperty('context');
    });
  });
});
