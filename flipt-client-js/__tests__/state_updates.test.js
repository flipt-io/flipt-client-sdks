import { jest } from '@jest/globals';

const { FliptClient: NodeClient } = await import('../src/node/index.ts');
const { FliptClient: BrowserClient } = await import('../src/browser/index.ts');
const { ErrorStrategy } = await import('../src/core/types.ts');

const doc = (...keys) => ({
  namespace: { key: 'default' },
  flags: keys.map((key) => ({ key, enabled: true, type: 'BOOLEAN_FLAG_TYPE' }))
});

const response = (etag, body) => ({
  ok: true,
  status: 200,
  headers: { get: (name) => (name.toLowerCase() === 'etag' ? etag : null) },
  json: async () => body
});

const makeLogger = () => ({
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
});

describe.each([
  ['node', NodeClient],
  ['browser', BrowserClient]
])('%s client state update listeners', (name, Client) => {
  let fetcher;
  let logger;
  let client;
  let n;
  let body;

  beforeEach(() => {
    n = 0;
    body = doc('a');
    // A new etag on every fetch, so the client always applies the body.
    fetcher = jest.fn(async () => response(`etag-${n++}`, body));
    logger = makeLogger();
  });

  afterEach(() => {
    client?.close();
    client = undefined;
  });

  const init = (opts = {}) =>
    Client.init({ fetcher, logger, updateInterval: 0, ...opts });

  test('is not called for the initial state', async () => {
    const onStateUpdate = jest.fn();
    client = await init({ onStateUpdate });
    expect(onStateUpdate).not.toHaveBeenCalled();
  });

  test('is called only when state changes', async () => {
    const onStateUpdate = jest.fn();
    client = await init({ onStateUpdate });

    expect(await client.refresh()).toBe(false);
    expect(onStateUpdate).not.toHaveBeenCalled();

    body = doc('a', 'b');
    expect(await client.refresh()).toBe(true);
    expect(onStateUpdate).toHaveBeenCalledTimes(1);

    expect(await client.refresh()).toBe(false);
    expect(onStateUpdate).toHaveBeenCalledTimes(1);
  });

  test('is not called on a failed fetch', async () => {
    const onStateUpdate = jest.fn();
    client = await init({
      onStateUpdate,
      errorStrategy: ErrorStrategy.Fallback
    });

    fetcher.mockRejectedValueOnce(new Error('boom'));
    expect(await client.refresh()).toBe(false);
    expect(onStateUpdate).not.toHaveBeenCalled();
  });

  test('does not keep the etag of a body that failed to apply', async () => {
    client = await init({ errorStrategy: ErrorStrategy.Fallback });

    fetcher.mockResolvedValueOnce({
      ...response('bad-etag', null),
      json: async () => {
        throw new Error('bad body');
      }
    });
    expect(await client.refresh()).toBe(false);

    await client.refresh();
    const lastOpts = fetcher.mock.calls[fetcher.mock.calls.length - 1][0];
    expect(lastOpts.etag).not.toBe('bad-etag');
  });

  test('can be added and removed at runtime', async () => {
    client = await init();
    const removed = jest.fn();
    const kept = jest.fn();
    const remove = client.addStateUpdateListener(removed);
    client.addStateUpdateListener(kept);

    body = doc('a', 'b');
    await client.refresh();
    expect(removed).toHaveBeenCalledTimes(1);
    expect(kept).toHaveBeenCalledTimes(1);

    remove();
    remove();

    body = doc('a');
    await client.refresh();
    expect(removed).toHaveBeenCalledTimes(1);
    expect(kept).toHaveBeenCalledTimes(2);
  });

  test('a throwing listener is logged and does not stop others', async () => {
    client = await init();
    const err = new Error('listener boom');
    const kept = jest.fn();
    client.addStateUpdateListener(() => {
      throw err;
    });
    client.addStateUpdateListener(kept);

    body = doc('a', 'b');
    await client.refresh();
    expect(kept).toHaveBeenCalledTimes(1);
    expect(logger.error).toHaveBeenCalledWith(
      'state update listener failed:',
      err
    );
  });

  test('a listener removed by an earlier listener is not called', async () => {
    client = await init();
    const second = jest.fn();
    let removeSecond;
    client.addStateUpdateListener(() => removeSecond());
    removeSecond = client.addStateUpdateListener(second);

    body = doc('a', 'b');
    await client.refresh();
    expect(second).not.toHaveBeenCalled();
  });

  test('no listener is called after close', async () => {
    client = await init();
    const second = jest.fn();
    client.addStateUpdateListener(() => client.close());
    client.addStateUpdateListener(second);

    body = doc('a', 'b');
    await client.refresh();
    expect(second).not.toHaveBeenCalled();

    const late = jest.fn();
    client.addStateUpdateListener(late);
    body = doc('a');
    await client.refresh();
    expect(late).not.toHaveBeenCalled();
  });
});
