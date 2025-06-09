import init, { Engine } from '../wasm/flipt_engine_wasm_js.js';
import wasm from '../wasm/flipt_engine_wasm_js_bg.wasm';
import { BaseFliptClient } from '../core/base';
import { ClientOptions, ClientOptionsFactory } from '../core/types';

export * from '../core/types';
export * from '../core/base';

export class FliptClient extends BaseFliptClient {
  /**
   * Initialize the client
   * @param options - optional client options
   * @returns {Promise<FliptClient>}
   */
  static async init(
    options: ClientOptions = ClientOptionsFactory.default()
  ): Promise<FliptClient> {
    const environment = options.environment ?? 'default';
    const namespace = options.namespace ?? 'default';

    let url = options.url ?? 'http://localhost:8080';
    url = url.replace(/\/$/, '');
    url = `${url}/internal/v1/evaluation/snapshot/namespace/${namespace}`;

    if (options.reference) {
      url = `${url}?reference=${options.reference}`;
    }

    const headers: Record<string, string> = {
      Accept: 'application/json',
      'x-flipt-accept-server-version': '1.47.0',
      'x-flipt-environment': environment
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
    const client = new FliptClient(engine, fetcher);
    client.storeEtag(resp);
    client.errorStrategy = options.errorStrategy;
    return client;
  }
}
