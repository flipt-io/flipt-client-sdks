import { BaseFliptClient } from './base';
import { ErrorStrategy, IFetcher, Response } from './types';

const engine = {
  snapshot: () => undefined,
  get_snapshot: () => ''
};

class TestClient extends BaseFliptClient {
  constructor(fetcher: IFetcher) {
    super(engine, fetcher);
  }

  setErrorStrategy(strategy: ErrorStrategy): void {
    this.errorStrategy = strategy;
  }
}

const response = (body: unknown, etag: string | null): Response => ({
  ok: true,
  status: 200,
  statusText: 'OK',
  headers: {
    get: (name: string) => (name.toLowerCase() === 'etag' ? etag : null)
  },
  json: async () => body
});

test('clears the stored etag when the response body fails to parse', async () => {
  const sent: (string | undefined)[] = [];
  let call = 0;

  const client = new TestClient(async (opts) => {
    sent.push(opts?.etag);
    call++;

    if (call === 1) {
      return {
        ...response({}, 'etag-1'),
        json: async () => {
          throw new Error('invalid json');
        }
      };
    }

    return response({ flags: [] }, 'etag-2');
  });

  client.setErrorStrategy(ErrorStrategy.Fail);

  await expect(client.refresh()).rejects.toThrow('invalid json');

  // The next request must not reuse the etag of the body that failed to parse,
  // otherwise the server answers 304 and the client stays on the old snapshot.
  await client.refresh();

  expect(sent).toEqual([undefined, undefined]);
});

test('keeps the etag when the response body parses', async () => {
  const sent: (string | undefined)[] = [];

  const client = new TestClient(async (opts) => {
    sent.push(opts?.etag);
    return response({ flags: [] }, 'etag-1');
  });

  await client.refresh();
  await client.refresh();

  expect(sent).toEqual([undefined, 'etag-1']);
});
