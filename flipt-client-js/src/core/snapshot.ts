import { ErrorStrategy, Response } from './types';

type SnapshotEngine = {
  seed_snapshot(snapshot: string): void;
  snapshot(data: unknown): void;
};

export async function initializeSnapshot({
  engine,
  snapshot,
  errorStrategy,
  fetcher,
  storeEtag
}: {
  engine: SnapshotEngine;
  snapshot?: string;
  errorStrategy?: ErrorStrategy;
  fetcher: () => Promise<Response>;
  storeEtag: (resp: Response) => void;
}): Promise<void> {
  if (snapshot) {
    engine.seed_snapshot(snapshot);
  }

  try {
    const resp = await fetcher();
    if (!resp.ok) {
      throw new Error(`Failed to fetch data: ${resp.statusText}`);
    }

    const data = await resp.json();
    engine.snapshot(data);
    storeEtag(resp);
  } catch (error) {
    if (!snapshot || errorStrategy !== ErrorStrategy.Fallback) {
      throw error;
    }
  }
}
