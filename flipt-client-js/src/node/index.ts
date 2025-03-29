import init, { Engine } from '../../dist/flipt_engine_wasm_js.js';
import wasm from '../../dist/flipt_engine_wasm_js_bg.wasm';
import { BaseFliptClient } from '../core/base';
import { ClientOptions, ErrorStrategy } from '../core/types';

export * from '../core/types';
export * from '../core/base';

export class FliptClient extends BaseFliptClient {
  private updateInterval?: NodeJS.Timeout;

  /**
   * Initialize the client
   * @param namespace - optional namespace to evaluate flags
   * @param options - optional client options
   * @returns {Promise<FliptClient>}
   */
  static async init(
    options: ClientOptions = {
      namespace: 'default',
      url: 'http://localhost:8080',
      reference: '',
      updateInterval: 120,
      errorStrategy: ErrorStrategy.Fail
    }
  ): Promise<FliptClient> {
    const namespace = options.namespace ?? 'default';

    let url = options.url ?? 'http://localhost:8080';
    url = url.replace(/\/$/, '');
    url = `${url}/internal/v1/evaluation/snapshot/namespace/${namespace}`;

    if (options.reference) {
      url = `${url}?reference=${options.reference}`;
    }

    const headers: Record<string, string> = {
      Accept: 'application/json',
      'x-flipt-accept-server-version': '1.47.0'
    };

    if (options.authentication) {
      if ('clientToken' in options.authentication) {
        headers['Authorization'] =
          `Bearer ${options.authentication.clientToken}`;
      } else if ('jwtToken' in options.authentication) {
        headers['Authorization'] = `JWT ${options.authentication.jwtToken}`;
      }
    }

    let fetcher = options.fetcher;

    if (!fetcher) {
      // Dynamically import node-fetch only when needed
      const { default: fetch } = await import('node-fetch');

      fetcher = async (opts?: { etag?: string }) => {
        if (opts && opts.etag) {
          headers['If-None-Match'] = opts.etag;
        }

        const resp = await fetch(url, {
          method: 'GET',
          headers
        });

        if (!resp.ok && resp.status !== 304) {
          throw new Error(`Failed to fetch data: ${resp.statusText}`);
        }

        return resp;
      };
    }

    // Initialize WASM engine
    // @ts-ignore
    await init(await wasm());

    if (!fetcher) {
      throw new Error('Failed to initialize fetcher');
    }

    // handle case if they pass in a custom fetcher that doesn't throw on non-2xx status codes
    const resp = await fetcher();
    if (!resp.ok) {
      throw new Error(`Failed to fetch data: ${resp.statusText}`);
    }

    const data = await resp.json();
    const engine = new Engine(namespace);
    engine.snapshot(data);

    // We know fetcher is defined here since we checked above
    const client = new FliptClient(engine, fetcher);
    client.storeEtag(resp);
    client.errorStrategy = options.errorStrategy;

    // Setup auto-refresh if interval is provided
    if (options.updateInterval && options.updateInterval > 0) {
      client.setupAutoRefresh(options.updateInterval * 1000);
    }

    return client;
  }

  private setupAutoRefresh(interval: number = 120_000): void {
    this.cleanupAutoRefresh();
    this.updateInterval = setInterval(async () => {
      await this.refresh();
    }, interval);
  }

  private cleanupAutoRefresh(): void {
    if (this.updateInterval) {
      clearInterval(this.updateInterval);
      this.updateInterval = undefined;
    }
  }

  public close() {
    this.cleanupAutoRefresh?.();
  }
}
