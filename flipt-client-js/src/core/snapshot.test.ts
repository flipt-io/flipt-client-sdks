import { initializeSnapshot } from './snapshot';
import { ErrorStrategy, Response } from './types';

const response = (body: unknown, ok = true): Response => ({
  ok,
  status: ok ? 200 : 500,
  statusText: ok ? 'OK' : 'Server Error',
  headers: { get: () => 'etag-1' },
  json: async () => body
});

test('seeds snapshot before fetching fresh state', async () => {
  const calls: string[] = [];
  const engine = {
    seed_snapshot: (snapshot: string) => calls.push(`seed:${snapshot}`),
    snapshot: (data: unknown) => calls.push(`fresh:${JSON.stringify(data)}`)
  };

  await initializeSnapshot({
    engine,
    snapshot: 'cached',
    errorStrategy: ErrorStrategy.Fallback,
    fetcher: async () => response({ flags: [] }),
    storeEtag: () => calls.push('etag')
  });

  expect(calls).toEqual(['seed:cached', 'fresh:{"flags":[]}', 'etag']);
});

test('allows offline init when snapshot and fallback strategy are provided', async () => {
  const calls: string[] = [];
  const engine = {
    seed_snapshot: (snapshot: string) => calls.push(`seed:${snapshot}`),
    snapshot: () => calls.push('fresh')
  };

  await expect(
    initializeSnapshot({
      engine,
      snapshot: 'cached',
      errorStrategy: ErrorStrategy.Fallback,
      fetcher: async () => {
        throw new Error('offline');
      },
      storeEtag: () => calls.push('etag')
    })
  ).resolves.toBeUndefined();

  expect(calls).toEqual(['seed:cached']);
});

test('throws offline init error without a snapshot', async () => {
  const engine = {
    seed_snapshot: () => undefined,
    snapshot: () => undefined
  };

  await expect(
    initializeSnapshot({
      engine,
      errorStrategy: ErrorStrategy.Fallback,
      fetcher: async () => {
        throw new Error('offline');
      },
      storeEtag: () => undefined
    })
  ).rejects.toThrow('offline');
});

test('throws offline init error with snapshot unless strategy is fallback', async () => {
  const engine = {
    seed_snapshot: () => undefined,
    snapshot: () => undefined
  };

  await expect(
    initializeSnapshot({
      engine,
      snapshot: 'cached',
      errorStrategy: ErrorStrategy.Fail,
      fetcher: async () => {
        throw new Error('offline');
      },
      storeEtag: () => undefined
    })
  ).rejects.toThrow('offline');
});
