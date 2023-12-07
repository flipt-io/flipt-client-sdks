import { FliptEvaluationClient } from '.';

const fliptUrl = process.env['FLIPT_URL'];
if (!fliptUrl) {
  console.error('please set the FLIPT_URL environment variable');
  process.exit(1);
}

const authToken = process.env['FLIPT_AUTH_TOKEN'];
if (!authToken) {
  console.error('please set the FLIPT_AUTH_TOKEN environment variable');
  process.exit(1);
}

test('variant', () => {
  const fec = new FliptEvaluationClient('default', {
    url: fliptUrl,
    auth_token: authToken
  });

  const variant = fec.evaluateVariant('flag1', 'someentity', { fizz: 'buzz' });

  expect(variant.error_message).toBeNull();
  expect(variant.status).toEqual('success');
  expect(variant.result).toBeDefined();
  expect(variant.result.flag_key).toEqual('flag1');
  expect(variant.result.match).toEqual(true);
  expect(variant.result.reason).toEqual('MATCH_EVALUATION_REASON');
  expect(variant.result.segment_keys).toContain('segment1');
  expect(variant.result.variant_key).toContain('variant1');
});

test('boolean', () => {
  const fec = new FliptEvaluationClient('default', {
    url: fliptUrl,
    auth_token: authToken
  });

  const boolean = fec.evaluateBoolean('flag_boolean', 'someentity', {
    fizz: 'buzz'
  });

  expect(boolean.error_message).toBeNull();
  expect(boolean.status).toEqual('success');
  expect(boolean.result).toBeDefined();
  expect(boolean.result.flag_key).toEqual('flag_boolean');
  expect(boolean.result.enabled).toEqual(true);
  expect(boolean.result.reason).toEqual('MATCH_EVALUATION_REASON');
});

test('variant failure', () => {
  const fec = new FliptEvaluationClient('default', {
    url: fliptUrl,
    auth_token: authToken
  });

  const variant = fec.evaluateVariant('nonexistent', 'someentity', {
    fizz: 'buzz'
  });

  expect(variant.result).toBeNull();
  expect(variant.status).toEqual('failure');
  expect(variant.error_message).toEqual(
    'failed to get flag information default/nonexistent'
  );
});

test('boolean', () => {
  const fec = new FliptEvaluationClient('default', {
    url: fliptUrl,
    auth_token: authToken
  });

  const boolean = fec.evaluateVariant('nonexistent', 'someentity', {
    fizz: 'buzz'
  });

  expect(boolean.result).toBeNull();
  expect(boolean.status).toEqual('failure');
  expect(boolean.error_message).toEqual(
    'failed to get flag information default/nonexistent'
  );
});
