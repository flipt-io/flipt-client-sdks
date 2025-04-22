import init, { Engine } from '../wasm/flipt_engine_wasm_js.js';
import { BaseFliptClient } from '~/core/base';
import { ClientOptions, ErrorStrategy } from '~/core/types';

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

export class FliptClient extends BaseFliptClient {
  /**
   * Initialize the client
   * @param options - optional client options
   * @param wasmOptions - options for loading WASM
   * @returns {Promise<FliptClient>}
   */
  static async init(
    options: ClientOptions = {
      namespace: 'default',
      url: 'http://localhost:8080',
      reference: '',
      updateInterval: 120,
      errorStrategy: ErrorStrategy.Fail
    },
    wasmOptions?: WasmOptions
  ): Promise<FliptClient> {
    if (!wasmOptions || !wasmOptions.wasm) {
      throw new Error(
        'WASM module must be provided in slim mode. Use the standard client or provide a wasm module.'
      );
    }

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

    // Initialize WASM engine with the provided WASM module
    await init(wasmOptions.wasm);

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

    // Create client instance
    const client = new FliptClient(engine, fetcher);
    client.storeEtag(resp);
    client.errorStrategy = options.errorStrategy;

    return client;
  }
}
