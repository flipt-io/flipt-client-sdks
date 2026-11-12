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
      'sse connection closed:',
      expect.any(Error)
    );
  });

  test('restarts the stream after the connection is closed permanently', async () => {
    jest.useFakeTimers();
    try {
      client = await init({ fetchMode: FetchMode.Streaming });
      expect(sources).toHaveLength(1);

      sources[0].readyState = 2;
      sources[0].onerror(new Error('gone'));

      await jest.advanceTimersByTimeAsync(2_000);

      expect(sources).toHaveLength(2);
      expect(sources[1].url).toBe(sources[0].url);

      // The new stream refreshes the flag state once it connects.
      const before = fetcher.mock.calls.length;
      sources[1].onopen({});
      await jest.advanceTimersByTimeAsync(1);
      expect(fetcher.mock.calls.length).toBe(before + 1);
    } finally {
      jest.useRealTimers();
    }
  });

  test('does not restart the stream after close()', async () => {
    jest.useFakeTimers();
    try {
      client = await init({ fetchMode: FetchMode.Streaming });

      sources[0].readyState = 2;
      sources[0].onerror(new Error('gone'));
      client.close();

      await jest.advanceTimersByTimeAsync(5_000);

      expect(sources).toHaveLength(1);
    } finally {
      jest.useRealTimers();
    }
  });

  test('schedules a single restart for repeated errors', async () => {
    jest.useFakeTimers();
    try {
      client = await init({ fetchMode: FetchMode.Streaming });

      sources[0].readyState = 2;
      sources[0].onerror(new Error('gone'));
      sources[0].onerror(new Error('gone again'));

      await jest.advanceTimersByTimeAsync(2_000);

      expect(sources).toHaveLength(2);
    } finally {
      jest.useRealTimers();
    }
  });

  test('backs off between restart attempts', async () => {
    jest.useFakeTimers();
    try {
      client = await init({ fetchMode: FetchMode.Streaming });

      sources[0].readyState = 2;
      sources[0].onerror(new Error('gone'));
      await jest.advanceTimersByTimeAsync(2_000);
      expect(sources).toHaveLength(2);

      // The restarted stream fails as well, so the next attempt waits longer.
      sources[1].readyState = 2;
      sources[1].onerror(new Error('gone again'));

      const delays = logger.warn.mock.calls
        .map((call) => call[0])
        .filter(
          (msg) =>
            typeof msg === 'string' &&
            msg.startsWith('sse connection closed, restarting in')
        )
        .map((msg) => Number(msg.match(/in (\d+)ms/)[1]));

      expect(delays).toHaveLength(2);
      expect(delays[0]).toBeGreaterThanOrEqual(500);
      expect(delays[0]).toBeLessThan(1_000);
      expect(delays[1]).toBeGreaterThanOrEqual(1_000);
      expect(delays[1]).toBeLessThan(2_000);
    } finally {
      jest.useRealTimers();
    }
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

  test('the restarted stream keeps the auth headers', async () => {
    jest.useFakeTimers();
    try {
      const client = await NodeClient.init({
        fetcher: async () => snapshotResponse('e'),
        fetchMode: FetchMode.Streaming,
        authentication: { clientToken: 'secret' }
      });

      sources[0].readyState = 2;
      sources[0].onerror(new Error('gone'));
      await jest.advanceTimersByTimeAsync(2_000);

      expect(sources).toHaveLength(2);
      expect(sources[1].init.fetch).toBeDefined();
      client.close();
    } finally {
      jest.useRealTimers();
    }
  });
});
