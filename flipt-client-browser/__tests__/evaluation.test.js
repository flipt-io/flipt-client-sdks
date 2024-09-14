const flipt = require('../dist/index.cjs');

describe('FliptEvaluationClient', () => {
  let fliptUrl;
  let authToken;
  let client;

  beforeAll(() => {
    fliptUrl = process.env['FLIPT_URL'];
    if (!fliptUrl) {
      console.error('please set the FLIPT_URL environment variable');
      process.exit(1);
    }

    authToken = process.env['FLIPT_AUTH_TOKEN'];
    if (!authToken) {
      console.error('please set the FLIPT_AUTH_TOKEN environment variable');
      process.exit(1);
    }
  });

  beforeAll(async () => {
    client = await flipt.FliptEvaluationClient.init('default', {
      url: fliptUrl,
      authentication: {
        clientToken: authToken
      }
    });
  });

  test('variant', () => {
    const variant = client.evaluateVariant('flag1', 'someentity', {
      fizz: 'buzz'
    });

    expect(variant.flagKey).toEqual('flag1');
    expect(variant.match).toEqual(true);
    expect(variant.reason).toEqual('MATCH_EVALUATION_REASON');
    expect(variant.segmentKeys).toContain('segment1');
    expect(variant.variantKey).toContain('variant1');
  });

  test('boolean', () => {
    const boolean = client.evaluateBoolean('flag_boolean', 'someentity', {
      fizz: 'buzz'
    });

    expect(boolean.enabled).toEqual(true);
    expect(boolean.reason).toEqual('MATCH_EVALUATION_REASON');
  });

  test('variant failure', () => {
    expect(() => {
      client.evaluateVariant('nonexistent', 'someentity', {
        fizz: 'buzz'
      });
    }).toThrow(
      'invalid request: failed to get flag information default/nonexistent'
    );
  });

  test('boolean failure', () => {
    expect(() => {
      client.evaluateVariant('nonexistent', 'someentity', {
        fizz: 'buzz'
      });
    }).toThrow(
      'invalid request: failed to get flag information default/nonexistent'
    );
  });

  test('refresh', async () => {
    await client.refresh();

    const variant = client.evaluateVariant('flag1', 'someentity', {
      fizz: 'buzz'
    });

    expect(variant.flagKey).toEqual('flag1');
    expect(variant.match).toEqual(true);
    expect(variant.reason).toEqual('MATCH_EVALUATION_REASON');
    expect(variant.segmentKeys).toContain('segment1');
    expect(variant.variantKey).toContain('variant1');
  });

  test('batch', () => {
    const batch = client.evaluateBatch([
      {
        flagKey: 'flag1',
        entityId: 'someentity',
        context: {
          fizz: 'buzz'
        }
      },
      {
        flagKey: 'flag_boolean',
        entityId: 'someentity',
        context: {
          fizz: 'buzz'
        }
      },
      {
        flagKey: 'notfound',
        entityId: 'someentity',
        context: {
          fizz: 'buzz'
        }
      }
    ]);

    expect(batch.responses).toHaveLength(3);
    const variant = batch.responses[0];

    expect(variant?.type).toEqual('VARIANT_EVALUATION_RESPONSE_TYPE');
    expect(variant?.variantEvaluationResponse?.flagKey).toEqual('flag1');
    expect(variant?.variantEvaluationResponse?.match).toEqual(true);
    expect(variant?.variantEvaluationResponse?.reason).toEqual(
      'MATCH_EVALUATION_REASON'
    );
    expect(variant?.variantEvaluationResponse?.segmentKeys).toContain(
      'segment1'
    );
    expect(variant?.variantEvaluationResponse?.variantKey).toContain(
      'variant1'
    );

    const boolean = batch.responses[1];
    expect(boolean?.type).toEqual('BOOLEAN_EVALUATION_RESPONSE_TYPE');
    expect(boolean?.booleanEvaluationResponse?.flagKey).toEqual('flag_boolean');
    expect(boolean?.booleanEvaluationResponse?.enabled).toEqual(true);
    expect(boolean?.booleanEvaluationResponse?.reason).toEqual(
      'MATCH_EVALUATION_REASON'
    );

    const error = batch.responses[2];
    expect(error?.type).toEqual('ERROR_EVALUATION_RESPONSE_TYPE');
    expect(error?.errorEvaluationResponse?.flagKey).toEqual('notfound');
    expect(error?.errorEvaluationResponse?.namespaceKey).toEqual('default');
    expect(error?.errorEvaluationResponse?.reason).toEqual(
      'NOT_FOUND_ERROR_EVALUATION_REASON'
    );
  });

  test('list flags', () => {
    const flags = client.listFlags();
    expect(flags).toBeDefined();
    expect(flags.length).toBe(2);
  });
});
