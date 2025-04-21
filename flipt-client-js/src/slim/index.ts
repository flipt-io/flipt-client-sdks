import init from '../wasm/flipt_engine_wasm_js.js';
import { BaseClient } from '~/core/base';
import { ClientOptions, ErrorStrategy, IFetcher } from '~/core/types';
import { Engine } from '../wasm/flipt_engine_wasm_js.js';

export * from '~/core/types';
export * from '~/core/base';

export interface WasmOptions {
  /**
   * The WASM module to use for evaluation.
   * Can be provided as:
   * - A URL string pointing to the WASM file
   * - A filesystem path to the WASM file (in Node.js environments)
   * - An ArrayBuffer or Uint8Array containing the WASM binary
   * - A WebAssembly.Module object
   *
   * The simplest way to provide the WASM module is to import it directly from the package:
   * ```
   * import wasm from '@flipt-io/flipt-client-js/flipt.wasm';
   *
   * const client = await FliptClient.init({ ... }, { wasm });
   * ```
   */
  wasm: ArrayBuffer | Uint8Array | WebAssembly.Module | string;
}

export class FliptClient extends BaseClient {
  /**
   * Initialize the client
   * @param options - optional client options
   * @param wasmOptions - optional WASM options
   * @returns {Promise<FliptClient>}
   */
  static async init(
    options: ClientOptions = {
      namespace: 'default',
      url: 'http://localhost:8080',
      reference: '',
      errorStrategy: ErrorStrategy.Fail
    },
    wasmOptions?: WasmOptions
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
      initWasm: async () => {
        if (wasmOptions?.wasm) {
          return await init(wasmOptions.wasm);
        }
        throw new Error('No WASM module provided');
      },
      createClient: (engine: Engine, fetcher: IFetcher) =>
        new FliptClient(engine, fetcher)
    });

    return client as FliptClient;
  }
}
