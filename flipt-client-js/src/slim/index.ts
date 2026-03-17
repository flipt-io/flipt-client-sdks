import init, { Engine } from '../wasm/flipt_engine_wasm_js.js';
import { BaseFliptClient } from '../core/base';
import { ClientOptions, ClientOptionsFactory } from '../core/types';
import { WasmEngine } from '../core/engines/wasm-engine';
import { RestEvaluationEngine } from '../core/engines/rest-engine';
import { JsLocalEngine } from '../core/engines/js-local-engine';

export * from '../core/types';
export * from '../core/base';

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
    options: ClientOptions = ClientOptionsFactory.default(),
    wasmOptions?: WasmOptions
  ): Promise<FliptClient> {
    const environment = options.environment ?? 'default';
    const namespace = options.namespace ?? 'default';
    const evaluationMode = options.evaluationMode ?? 'wasm';

    const baseUrl = (options.url ?? 'http://localhost:8080').replace(/\/$/, '');
    let url = `${baseUrl}/internal/v1/evaluation/snapshot/namespace/${namespace}`;

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

    if (!fetcher) {
      throw new Error('Failed to initialize fetcher');
    }

    let engine;
    let resp: Awaited<ReturnType<typeof fetcher>> | undefined;

    if (evaluationMode === 'rest') {
      engine = new RestEvaluationEngine(baseUrl, namespace, headers);
    } else if (evaluationMode === 'js-local') {
      engine = new JsLocalEngine(namespace);
      resp = await fetcher();
      if (!resp.ok) {
        throw new Error(`Failed to fetch data: ${resp.statusText}`);
      }
      const data = await resp.json();
      engine.snapshot(data);
    } else {
      if (!wasmOptions || !wasmOptions.wasm) {
        throw new Error(
          'WASM module must be provided in slim mode. Use the standard client or provide a wasm module.'
        );
      }

      try {
        // Initialize WASM engine with the provided WASM module
        await init(wasmOptions.wasm);

        // handle case if they pass in a custom fetcher that doesn't throw on non-2xx status codes
        resp = await fetcher();
        if (!resp.ok) {
          throw new Error(`Failed to fetch data: ${resp.statusText}`);
        }

        const data = await resp.json();
        const wasmEngineInstance = new Engine(namespace);
        engine = new WasmEngine(wasmEngineInstance, namespace);
        engine.snapshot(data);
      } catch (error) {
        if (!options.enableAutoFallback) {
          throw error;
        }
        // Fallback: wasm → js-local → rest
        try {
          engine = new JsLocalEngine(namespace);
          resp = await fetcher();
          if (!resp.ok) {
            throw new Error(`Failed to fetch data: ${resp.statusText}`);
          }
          const data = await resp.json();
          engine.snapshot(data);
        } catch {
          engine = new RestEvaluationEngine(baseUrl, namespace, headers);
        }
      }
    }

    // Create client instance
    const client = new FliptClient(engine, fetcher);
    if (resp) {
      client.storeEtag(resp);
    }
    client.errorStrategy = options.errorStrategy;
    client.hook = options.hook;

    return client;
  }
}
