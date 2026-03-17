import init, { Engine } from '../wasm/flipt_engine_wasm_js.js';
import wasm from '../wasm/flipt_engine_wasm_js_bg.wasm';
import { BaseFliptClient } from '../core/base';
import { ClientOptions, ClientOptionsFactory } from '../core/types';
import { WasmEngine } from '../core/engines/wasm-engine';
import { RestEvaluationEngine } from '../core/engines/rest-engine';
import { JsLocalEngine } from '../core/engines/js-local-engine';

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
    options: ClientOptions = ClientOptionsFactory.default()
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
      try {
        // Initialize WASM engine
        // @ts-ignore
        await init(await wasm());

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

    // We know fetcher is defined here since we checked above
    const client = new FliptClient(engine, fetcher);
    if (resp) {
      client.storeEtag(resp);
    }
    client.errorStrategy = options.errorStrategy;
    client.hook = options.hook;

    // Setup auto-refresh if interval is provided
    if (
      evaluationMode !== 'rest' &&
      options.updateInterval &&
      options.updateInterval > 0
    ) {
      client.setupAutoRefresh(options.updateInterval * 1_000);
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
