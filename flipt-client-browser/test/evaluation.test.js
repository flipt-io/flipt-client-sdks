const FliptEvaluationClient = require('../lib/index.js');

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
    client = await FliptEvaluationClient.init('default', {
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

    expect(variant.error_message).toBeUndefined();
    expect(variant.status).toEqual('success');
    expect(variant.result).toBeDefined();
    expect(variant.result?.flag_key).toEqual('flag1');
    expect(variant.result?.match).toEqual(true);
    expect(variant.result?.reason).toEqual('MATCH_EVALUATION_REASON');
    expect(variant.result?.segment_keys).toContain('segment1');
    expect(variant.result?.variant_key).toContain('variant1');
  });

  test('boolean', () => {
    const boolean = client.evaluateBoolean('flag_boolean', 'someentity', {
      fizz: 'buzz'
    });

    expect(boolean.error_message).toBeUndefined();
    expect(boolean.status).toEqual('success');
    expect(boolean.result).toBeDefined();
    expect(boolean.result?.flag_key).toEqual('flag_boolean');
    expect(boolean.result?.enabled).toEqual(true);
    expect(boolean.result?.reason).toEqual('MATCH_EVALUATION_REASON');
  });

  test('variant failure', () => {
    const variant = client.evaluateVariant('nonexistent', 'someentity', {
      fizz: 'buzz'
    });

    expect(variant.result).toBeUndefined();
    expect(variant.status).toEqual('failure');
    expect(variant.error_message).toEqual(
      'invalid request: failed to get flag information default/nonexistent'
    );
  });

  test('boolean failure', () => {
    const boolean = client.evaluateVariant('nonexistent', 'someentity', {
      fizz: 'buzz'
    });

    expect(boolean.result).toBeUndefined();
    expect(boolean.status).toEqual('failure');
    expect(boolean.error_message).toEqual(
      'invalid request: failed to get flag information default/nonexistent'
    );
  });

  test('refresh', async () => {
    await client.refresh();

    const variant = client.evaluateVariant('flag1', 'someentity', {
      fizz: 'buzz'
    });

    expect(variant.error_message).toBeUndefined();
    expect(variant.status).toEqual('success');
    expect(variant.result).toBeDefined();
    expect(variant.result?.flag_key).toEqual('flag1');
    expect(variant.result?.match).toEqual(true);
    expect(variant.result?.reason).toEqual('MATCH_EVALUATION_REASON');
    expect(variant.result?.segment_keys).toContain('segment1');
    expect(variant.result?.variant_key).toContain('variant1');
  });
});
