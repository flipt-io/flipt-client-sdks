import assert from 'node:assert/strict';
import { ErrorStrategy, FliptClient } from '../dist/node/index.mjs';

const url = process.env.FLIPT_URL;
if (!url) {
  console.log('skipping snapshot integration: FLIPT_URL not set');
  process.exit(0);
}

const authToken = process.env.FLIPT_AUTH_TOKEN;
const options = {
  url,
  authentication: authToken ? { clientToken: authToken } : undefined
};

const online = await FliptClient.init(options);
try {
  const snapshot = online.getSnapshot();
  assert.ok(snapshot, 'expected exported snapshot');

  const offline = await FliptClient.init({
    snapshot,
    errorStrategy: ErrorStrategy.Fallback,
    fetcher: async () => {
      throw new Error('offline');
    }
  });

  try {
    const variant = offline.evaluateVariant({
      flagKey: 'flag1',
      entityId: 'someentity',
      context: { fizz: 'buzz' }
    });

    assert.equal(variant.match, true);
    assert.equal(variant.flagKey, 'flag1');
    assert.equal(variant.reason, 'MATCH_EVALUATION_REASON');
    assert.equal(variant.variantKey, 'variant1');
  } finally {
    offline.close();
  }
} finally {
  online.close();
}
