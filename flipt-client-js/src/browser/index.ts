import init from '../wasm/flipt_engine_wasm_js.js';
import wasm from '../wasm/flipt_engine_wasm_js_bg.wasm';
import { BaseClient } from '~/core/base';
import { ClientOptions, ErrorStrategy, IFetcher } from '~/core/types';
import { Engine } from '../wasm/flipt_engine_wasm_js.js';

export * from '~/core/types';
export * from '~/core/base';

export class FliptClient extends BaseClient {
  /**
   * Initialize the client
   * @param options - optional client options
   * @returns {Promise<FliptClient>}
   */
  static async init(
    options: ClientOptions = {
      namespace: 'default',
      url: 'http://localhost:8080',
      reference: '',
      errorStrategy: ErrorStrategy.Fail
    }
  ): Promise<FliptClient> {
    if (!options.fetcher) {
      options.fetcher = async (opts?: { etag?: string }) => {
        const headers: Record<string, string> = {
          Accept: 'application/json',
          'x-flipt-accept-server-version': '1.47.0'
        };

        if (opts?.etag) {
          headers['If-None-Match'] = opts.etag;
        }

        if (options.authentication) {
          if ('clientToken' in options.authentication) {
            headers['Authorization'] =
              `Bearer ${options.authentication.clientToken}`;
          } else if ('jwtToken' in options.authentication) {
            headers['Authorization'] = `JWT ${options.authentication.jwtToken}`;
          }
        }

        let url = options.url ?? 'http://localhost:8080';
        url = url.replace(/\/$/, '');
        url = `${url}/internal/v1/evaluation/snapshot/namespace/${options.namespace ?? 'default'}`;

        if (options.reference) {
          url = `${url}?reference=${options.reference}`;
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

    const client = await BaseClient.initialize({
      options,
      // @ts-ignore
      initWasm: async () => await init(await wasm()),
      createClient: (engine: Engine, fetcher: IFetcher) =>
        new FliptClient(engine, fetcher)
    });

    return client as FliptClient;
  }
}
