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

  beforeEach(async () => {
    client = await flipt.FliptEvaluationClient.init('default', {
      url: fliptUrl,
      authentication: {
        client_token: authToken
      }
    });
  });

  test('variant', () => {
    const variant = client.evaluateVariant('flag1', 'someentity', {
      fizz: 'buzz'
    });

    expect(variant.flag_key).toEqual('flag1');
    expect(variant.match).toEqual(true);
    expect(variant.reason).toEqual('MATCH_EVALUATION_REASON');
    expect(variant.segment_keys).toContain('segment1');
    expect(variant.variant_key).toContain('variant1');
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
    }).toThrow('invalid request: failed to get flag information default/nonexistent');
  });

  test('boolean failure', () => {
    expect(() => {
      client.evaluateVariant('nonexistent', 'someentity', {
        fizz: 'buzz'
      });
    }).toThrow('invalid request: failed to get flag information default/nonexistent');
  });

  test('refresh', async () => {
    await client.refresh();

    const variant = client.evaluateVariant('flag1', 'someentity', {
      fizz: 'buzz'
    });

    expect(variant.flag_key).toEqual('flag1');
    expect(variant.match).toEqual(true);
    expect(variant.reason).toEqual('MATCH_EVALUATION_REASON');
    expect(variant.segment_keys).toContain('segment1');
    expect(variant.variant_key).toContain('variant1');
  });

  test('batch', () => {
    const batch = client.evaluateBatch([
      {
        flag_key: 'flag1',
        entity_id: 'someentity',
        context: {
          fizz: 'buzz'
        }
      },
      {
        flag_key: 'flag_boolean',
        entity_id: 'someentity',
        context: {
          fizz: 'buzz'
        }
      },
      {
        flag_key: 'notfound',
        entity_id: 'someentity',
        context: {
          fizz: 'buzz'
        }
      }
    ]);

    expect(batch.responses).toHaveLength(3);
    const variant = batch.responses[0];

    expect(variant?.type).toEqual('VARIANT_EVALUATION_RESPONSE_TYPE');
    expect(variant?.variant_evaluation_response?.flag_key).toEqual('flag1');
    expect(variant?.variant_evaluation_response?.match).toEqual(true);
    expect(variant?.variant_evaluation_response?.reason).toEqual(
      'MATCH_EVALUATION_REASON'
    );
    expect(variant?.variant_evaluation_response?.segment_keys).toContain(
      'segment1'
    );
    expect(variant?.variant_evaluation_response?.variant_key).toContain(
      'variant1'
    );

    const boolean = batch.responses[1];
    expect(boolean?.type).toEqual('BOOLEAN_EVALUATION_RESPONSE_TYPE');
    expect(boolean?.boolean_evaluation_response?.flag_key).toEqual(
      'flag_boolean'
    );
    expect(boolean?.boolean_evaluation_response?.enabled).toEqual(true);
    expect(boolean?.boolean_evaluation_response?.reason).toEqual(
      'MATCH_EVALUATION_REASON'
    );

    const error = batch.responses[2];
    expect(error?.type).toEqual('ERROR_EVALUATION_RESPONSE_TYPE');
    expect(error?.error_evaluation_response?.flag_key).toEqual('notfound');
    expect(error?.error_evaluation_response?.namespace_key).toEqual('default');
    expect(error?.error_evaluation_response?.reason).toEqual(
      'NOT_FOUND_ERROR_EVALUATION_REASON'
    );
  });
});
