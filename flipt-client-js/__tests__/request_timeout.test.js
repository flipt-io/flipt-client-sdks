import { jest } from '@jest/globals';
import http from 'node:http';

const { FliptClient: NodeClient } = await import('../src/node/index.ts');
const { FliptClient: BrowserClient } = await import('../src/browser/index.ts');
const { FliptClient: SlimClient } = await import('../src/slim/index.ts');
const { ErrorStrategy } = await import('../src/core/types.ts');

const body = JSON.stringify({ namespace: { key: 'default' }, flags: [] });

describe.each([
  ['node', (opts) => NodeClient.init(opts)],
  ['browser', (opts) => BrowserClient.init(opts)],
  ['slim', (opts) => SlimClient.init(opts, { wasm: new Uint8Array() })]
])('%s client request timeout', (_name, initClient) => {
  let server;
  let url;
  let respond;
  let client;

  const ok = (res) => res.end(body);
  const neverResponds = () => {};
  const neverFinishesResponding = (res) => res.write(body.slice(0, 10));

  beforeEach(async () => {
    respond = ok;
    server = http.createServer((_req, res) => respond(res));
    await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
    url = `http://127.0.0.1:${server.address().port}`;
  });

  afterEach(async () => {
    jest.restoreAllMocks();
    client?.close();
    client = undefined;
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
  });

  const init = (opts = {}) => initClient({ url, ...opts });

  test('init succeeds when Flipt responds within the timeout', async () => {
    client = await init({ requestTimeout: 5 });

    expect(client).toBeDefined();
  });

  test('init fails when Flipt does not respond within the timeout', async () => {
    respond = neverResponds;

    await expect(init({ requestTimeout: 0.5 })).rejects.toThrow('aborted');
  });

  test('init fails when Flipt does not finish responding within the timeout', async () => {
    respond = neverFinishesResponding;

    await expect(init({ requestTimeout: 0.5 })).rejects.toThrow('aborted');
  });

  test('a refresh that times out does not stop later refreshes', async () => {
    client = await init({
      requestTimeout: 0.5,
      errorStrategy: ErrorStrategy.Fail
    });

    respond = neverResponds;
    await expect(client.refresh()).rejects.toThrow('aborted');

    respond = ok;
    await expect(client.refresh()).resolves.toBeDefined();
  });

  test('defaults to 30 seconds', async () => {
    const timeout = jest.spyOn(AbortSignal, 'timeout');

    client = await init();

    expect(timeout).toHaveBeenCalledWith(30_000);
  });
});
