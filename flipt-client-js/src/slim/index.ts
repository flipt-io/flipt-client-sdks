import init from '../wasm/flipt_engine_wasm_js.js';
import { BaseFliptClient } from '~/core/base';
import { ClientOptions, ErrorStrategy, IFetcher } from '~/core/types';
import { Engine } from '../wasm/flipt_engine_wasm_js.js';

export * from '~/core/types';
export * from '~/core/base';

export interface WasmOptions {
  /**
   * URL to the WASM module
   */
  url?: string;
  /**
   * Path to the WASM module on the filesystem
   */
  path?: string;
  /**
   * ArrayBuffer containing the WASM module
   */
  buffer?: ArrayBuffer;
  /**
   * WebAssembly.Module instance
   */
  module?: WebAssembly.Module;
}

export class FliptClient extends BaseFliptClient {
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
            headers['Authorization'] = `Bearer ${options.authentication.clientToken}`;
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

    let wasmModule: WebAssembly.Module | undefined;

    if (wasmOptions?.module) {
      wasmModule = wasmOptions.module;
    } else if (wasmOptions?.buffer) {
      wasmModule = await WebAssembly.compile(wasmOptions.buffer);
    } else if (wasmOptions?.url) {
      const response = await fetch(wasmOptions.url);
      const buffer = await response.arrayBuffer();
      wasmModule = await WebAssembly.compile(buffer);
    } else if (wasmOptions?.path) {
      // Node.js environment
      const fs = await import('fs/promises');
      const buffer = await fs.readFile(wasmOptions.path);
      wasmModule = await WebAssembly.compile(buffer);
    }

    const client = await BaseFliptClient.initialize({
      options,
      initWasm: async () => {
        if (wasmModule) {
          return await init(wasmModule);
        }
        throw new Error('No WASM module provided');
      },
      createClient: (engine: Engine, fetcher: IFetcher) => new FliptClient(engine, fetcher)
    });

    return client as FliptClient;
  }
}
