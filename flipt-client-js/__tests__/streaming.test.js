import { jest } from '@jest/globals';

// Registry of fake EventSource instances created by the clients under test.
const sources = [];

class FakeEventSource {
  constructor(url, init) {
    this.url = url;
    this.init = init;
    this.readyState = 1;
    this.onopen = null;
    this.onmessage = null;
    this.onerror = null;
    this.close = jest.fn(() => {
      this.readyState = 2;
    });
    sources.push(this);
  }
}

jest.unstable_mockModule('eventsource', () => ({
  EventSource: FakeEventSource
}));

const { FliptClient: NodeClient } = await import('../src/node/index.ts');
const { FliptClient: BrowserClient } = await import('../src/browser/index.ts');
const { FetchMode, ErrorStrategy } = await import('../src/core/types.ts');

const snapshotResponse = (etag) => ({
  ok: true,
  status: 200,
  headers: { get: (name) => (name.toLowerCase() === 'etag' ? etag : null) },
  json: async () => ({ namespace: { key: 'default' }, flags: [] })
});

const flush = () => new Promise((resolve) => setImmediate(resolve));

const makeLogger = () => ({
  debug: jest.fn(),
  info: jest.fn(),
  warn: jest.fn(),
  error: jest.fn()
});

describe.each([
  ['node', NodeClient],
  ['browser', BrowserClient]
])('%s client streaming', (name, Client) => {
  let originalEventSource;
  let fetcher;
  let logger;
  let client;

  beforeEach(() => {
    sources.length = 0;
    originalEventSource = globalThis.EventSource;
    // The browser client uses the global EventSource.
    globalThis.EventSource = FakeEventSource;
    let n = 0;
    fetcher = jest.fn(async () => snapshotResponse(`etag-${n++}`));
    logger = makeLogger();
  });

  afterEach(() => {
    client?.close();
    client = undefined;
    globalThis.EventSource = originalEventSource;
  });

  const init = (opts = {}) =>
    Client.init({
      url: 'http://flipt.test/',
      environment: 'prod',
      namespace: 'ns',
      fetcher,
      logger,
      ...opts
    });

  test('does not open a stream by default', async () => {
    client = await init({ updateInterval: 0 });
    expect(sources).toHaveLength(0);
  });

  test('opens a stream at the v2 endpoint with the reference', async () => {
    client = await init({
      fetchMode: FetchMode.Streaming,
      reference: 'abc123'
    });

    expect(sources).toHaveLength(1);
    expect(sources[0].url).toBe(
      'http://flipt.test/client/v2/environments/prod/namespaces/ns/stream?reference=abc123'
    );
  });

  test('refreshes on refetchEvaluation events', async () => {
    client = await init({ fetchMode: FetchMode.Streaming });
    const before = fetcher.mock.calls.length;

    sources[0].onmessage({
      data: JSON.stringify({ type: 'refetchEvaluation' })
    });
    await flush();

    expect(fetcher.mock.calls.length).toBe(before + 1);
  });

  test('ignores unrelated events', async () => {
    client = await init({ fetchMode: FetchMode.Streaming });
    const before = fetcher.mock.calls.length;

    sources[0].onmessage({ data: JSON.stringify({ type: 'ping' }) });
    await flush();

    expect(fetcher.mock.calls.length).toBe(before);
  });

  test('logs and survives unparseable events', async () => {
    client = await init({ fetchMode: FetchMode.Streaming });
    const before = fetcher.mock.calls.length;

    sources[0].onmessage({ data: 'not json' });
    await flush();

    expect(logger.warn).toHaveBeenCalledWith('sse parse error:', 'not json');
    expect(fetcher.mock.calls.length).toBe(before);
  });

  test('refreshes when the stream (re)connects', async () => {
    client = await init({ fetchMode: FetchMode.Streaming });
    const before = fetcher.mock.calls.length;

    sources[0].onopen({});
    await flush();

    expect(fetcher.mock.calls.length).toBe(before + 1);
  });

  test('a failed refresh from the stream is logged, not thrown', async () => {
    client = await init({
      fetchMode: FetchMode.Streaming,
      errorStrategy: ErrorStrategy.Fail
    });
    fetcher.mockRejectedValueOnce(new Error('boom'));

    sources[0].onmessage({
      data: JSON.stringify({ type: 'refetchEvaluation' })
    });
    await flush();

    expect(logger.warn).toHaveBeenCalledWith(
      'sse refresh failed:',
      expect.any(Error)
    );
  });

  test('distinguishes transient errors from a closed stream', async () => {
    client = await init({ fetchMode: FetchMode.Streaming });

    sources[0].onerror(new Error('blip'));
    expect(logger.warn).toHaveBeenCalledWith(
      'sse error, reconnecting:',
      expect.any(Error)
    );
    expect(logger.error).not.toHaveBeenCalled();

    sources[0].readyState = 2;
    sources[0].onerror(new Error('gone'));
    expect(logger.error).toHaveBeenCalledWith(
      'sse connection closed, no further updates:',
      expect.any(Error)
    );
  });

  test('close() closes the stream once', async () => {
    client = await init({ fetchMode: FetchMode.Streaming });

    client.close();
    client.close();

    expect(sources[0].close).toHaveBeenCalledTimes(1);
  });
});

describe('node client streaming specifics', () => {
  beforeEach(() => {
    sources.length = 0;
  });

  test('does not start polling in streaming mode', async () => {
    jest.useFakeTimers();
    try {
      const fetcher = jest.fn(async () => snapshotResponse('e'));
      const client = await NodeClient.init({
        fetcher,
        fetchMode: FetchMode.Streaming,
        updateInterval: 1
      });
      const before = fetcher.mock.calls.length;

      jest.advanceTimersByTime(5_000);

      expect(fetcher.mock.calls.length).toBe(before);
      client.close();
    } finally {
      jest.useRealTimers();
    }
  });

  test('sends auth headers and the SSE accept header on the stream', async () => {
    const realFetch = globalThis.fetch;
    const mockFetch = jest.fn(async () => ({}));
    globalThis.fetch = mockFetch;
    try {
      const client = await NodeClient.init({
        fetcher: async () => snapshotResponse('e'),
        fetchMode: FetchMode.Streaming,
        authentication: { clientToken: 'secret' }
      });

      await sources[0].init.fetch('http://x/stream', { headers: { a: 'b' } });

      const [, init] = mockFetch.mock.calls[0];
      expect(init.headers).toMatchObject({
        a: 'b',
        Accept: 'text/event-stream',
        Authorization: 'Bearer secret'
      });
      client.close();
    } finally {
      globalThis.fetch = realFetch;
    }
  });
});
