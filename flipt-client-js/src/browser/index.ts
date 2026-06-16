import init, { Engine } from '../wasm/flipt_engine_wasm_js.js';
import wasm from '../wasm/flipt_engine_wasm_js_bg.wasm';
import { BaseFliptClient } from '../core/base';
import { initializeSnapshot } from '../core/snapshot';
import { ClientOptions, ClientOptionsFactory, FetchMode } from '../core/types';

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
    const fetchMode = options.fetchMode ?? FetchMode.Polling;

    let baseUrl = options.url ?? 'http://localhost:8080';
    baseUrl = baseUrl.replace(/\/$/, '');
    let url = `${baseUrl}/internal/v1/evaluation/snapshot/namespace/${namespace}`;

    if (options.reference) {
      url = `${url}?reference=${options.reference}`;
    }

    let streamUrl = `${baseUrl}/client/v2/environments/${environment}/namespaces/${namespace}/stream`;

    if (options.reference) {
      streamUrl = `${streamUrl}?reference=${options.reference}`;
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

    const engine = new Engine(namespace);
    const client = new FliptClient(engine, fetcher);
    await initializeSnapshot({
      engine,
      snapshot: options.snapshot,
      errorStrategy: options.errorStrategy,
      fetcher,
      storeEtag: (resp) => client.storeEtag(resp)
    });
    client.errorStrategy = options.errorStrategy;
    client.hook = options.hook;
    client.logger = options.logger ?? client.logger;

    // Setup data refresh based on fetch mode
    if (fetchMode === FetchMode.Streaming) {
      client.setupStream(streamUrl);
    }

    return client;
  }

  private setupStream(url: string): void {
    // Native browser EventSource does not support custom headers.
    // For authenticated SSE, use cookies or configure the server accordingly.
    const eventSource = new EventSource(url);

    eventSource.onmessage = (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data);
        this.logger.debug('sse message:', data);
        if (data.type === 'refetchEvaluation') {
          this.refresh().catch((err) => {
            this.logger.warn('sse refresh failed:', err);
          });
        }
      } catch {
        this.logger.warn('sse parse error:', event.data);
      }
    };

    eventSource.onerror = (err) => {
      this.logger.warn('sse error:', err);
    };

    this.eventSource = eventSource;
  }
}
