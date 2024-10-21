import { Engine } from '../dist/flipt_engine_wasm.js';
import { serialize, deserialize } from './utils';

import {
  AuthenticationStrategy,
  ClientOptions,
  EvaluationRequest,
  IFetcher,
  VariantEvaluationResponse,
  BooleanEvaluationResponse,
  BatchEvaluationResponse,
  ErrorEvaluationResponse,
  EvaluationResponse,
  Flag,
  IFetcherOptions
} from './models';

import {
  VariantResult,
  BooleanResult,
  BatchResult,
  ListFlagsResult,
  StreamChunk
} from './internal/models';

export class FliptEvaluationClient {
  private namespace: string;
  private engine: Engine;
  private fetcher: IFetcher;

  private etag?: string;
  private updateInterval?: NodeJS.Timeout;
  private abortController?: AbortController;
  private isStreaming: boolean = false;

  private constructor(namespace: string, engine: Engine, fetcher: IFetcher) {
    this.namespace = namespace;
    this.engine = engine;
    this.fetcher = fetcher;
  }

  /**
   * Initialize the client
   * @param namespace - optional namespace to evaluate flags
   * @param options - optional client options
   * @returns {Promise<FliptEvaluationClient>}
   */
  static async init(
    namespace: string = 'default',
    options: ClientOptions<AuthenticationStrategy> = {
      url: 'http://localhost:8080',
      reference: '',
      updateInterval: 120,
      fetchMode: 'polling'
    }
  ): Promise<FliptEvaluationClient> {
    let baseUrl = options.url ?? 'http://localhost:8080';
    baseUrl = baseUrl.replace(/\/$/, '');

    let url = `${baseUrl}/internal/v1/evaluation/snapshot/namespace/${namespace}`;
    if (options.reference) {
      url = `${url}?reference=${options.reference}`;
    }

    const headers = new Headers();
    headers.append('Accept', 'application/json');
    headers.append('x-flipt-accept-server-version', '1.47.0');

    if (options.authentication) {
      if ('clientToken' in options.authentication) {
        headers.append(
          'Authorization',
          `Bearer ${options.authentication.clientToken}`
        );
      } else if ('jwtToken' in options.authentication) {
        headers.append(
          'Authorization',
          `JWT ${options.authentication.jwtToken}`
        );
      }
    }

    let fetcher = options.fetcher;

    if (!fetcher) {
      fetcher = async (url: string, opts?: IFetcherOptions) => {
        if (opts && opts.etag) {
          headers.append('If-None-Match', opts.etag);
        }

        const resp = await fetch(url, {
          method: 'GET',
          headers,
          signal: opts?.signal
        });

        if (resp.status === 304) {
          return resp;
        }

        if (!resp.ok && resp.status !== 304) {
          throw new Error(`Failed to fetch data: ${resp.statusText}`);
        }

        return resp;
      };
    }

    // handle case if they pass in a custom fetcher that doesn't throw on non-2xx status codes
    // do initial fetch to snapshot url
    const resp = await fetcher(url);
    if (!resp.ok) {
      throw new Error(`Failed to fetch data: ${resp.statusText}`);
    }

    const data = await resp.json();
    const engine = new Engine(namespace);
    engine.snapshot(data);

    const client = new FliptEvaluationClient(namespace, engine, fetcher);
    client.storeEtag(resp);

    switch (options.fetchMode) {
      case 'polling':
        if (options.updateInterval && options.updateInterval > 0) {
          client.startPollingUpdates(url, options.updateInterval * 1000);
        }
        break;
      case 'streaming':
        url = `${baseUrl}/internal/v1/evaluation/snapshots?[]namespaces=${namespace}`;
        client.startStreamingUpdates(url);
        break;
    }

    return client;
  }

  /**
   * Store etag from response for next requests
   * @param resp - response to store etag from
   * @returns void
   */
  private storeEtag(resp: Response) {
    let etag = resp.headers.get('etag');
    if (etag) {
      this.etag = etag;
    }
  }

  /**
   * Start the auto refresh interval
   * @param url - url to refresh
   * @param interval - optional interval in milliseconds
   * @returns void
   */
  private startPollingUpdates(url: string, interval: number = 120_000) {
    this.stopPollingUpdates();
    this.updateInterval = setInterval(async () => {
      await this.refresh(url);
    }, interval);
  }

  /**
   * Stop the auto refresh interval
   * @returns void
   */
  private stopPollingUpdates() {
    if (this.updateInterval) {
      clearInterval(this.updateInterval);
      this.updateInterval = undefined;
    }
  }

  /**
   * Start the streaming updates
   * @param url - url to stream updates from
   * @returns void
   */
  private async startStreamingUpdates(url: string) {
    this.stopStreamingUpdates();
    this.abortController = new AbortController();
    this.isStreaming = true;

    try {
      const response = await this.fetcher(url, {
        signal: this.abortController.signal
      });
      const reader = response.body?.getReader();
      if (!reader) throw new Error('No response body');

      const decoder = new TextDecoder();
      let buffer = '';

      while (this.isStreaming) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.trim()) {
            const data = JSON.parse(line) as StreamChunk;
            this.engine.snapshot(data.result.namespaces[this.namespace]);
          }
        }
      }
    } catch (error) {
      // check if error has name property
      if (
        error &&
        typeof error === 'object' &&
        'name' in error &&
        error.name === 'AbortError'
      ) {
        // abort is expected when the client is closed
      } else {
        throw error;
      }
    } finally {
      this.isStreaming = false;
    }
  }

  /**
   * Stop the streaming updates
   * @returns void
   */
  private stopStreamingUpdates() {
    this.isStreaming = false;
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = undefined;
    }
  }

  /**
   * Refresh the flags snapshot
   * @param url - url to refresh
   * @returns {boolean} true if snapshot changed
   */
  public async refresh(url: string): Promise<boolean> {
    const opts = { etag: this.etag };
    const resp = await this.fetcher(url, opts);

    let etag = resp.headers.get('etag');
    if (this.etag && this.etag === etag) {
      return false;
    }

    this.storeEtag(resp);

    const data = await resp.json();
    this.engine.snapshot(data);
    return true;
  }

  /**
   * Evaluate a variant flag
   * @param flagKey - flag key to evaluate
   * @param entityId - entity id to evaluate
   * @param context - optional evaluation context
   * @returns {VariantEvaluationResponse}
   */
  public evaluateVariant(
    flagKey: string,
    entityId: string,
    context: {}
  ): VariantEvaluationResponse {
    const evaluationRequest: EvaluationRequest = {
      flagKey,
      entityId,
      context
    };

    const result: VariantResult | null = this.engine.evaluate_variant(
      serialize(evaluationRequest)
    );

    if (result === null) {
      throw new Error('Failed to evaluate variant');
    }

    const variantResult = deserialize<VariantResult>(result);

    if (variantResult.status === 'failure') {
      throw new Error(variantResult.errorMessage);
    }

    if (!variantResult.result) {
      throw new Error('Failed to evaluate variant');
    }

    return deserialize<VariantEvaluationResponse>(variantResult.result);
  }

  /**
   * Evaluate a boolean flag
   * @param flagKey - flag key to evaluate
   * @param entityId - entity id to evaluate
   * @param context - optional evaluation context
   * @returns {BooleanEvaluationResponse}
   */
  public evaluateBoolean(
    flagKey: string,
    entityId: string,
    context: {}
  ): BooleanEvaluationResponse {
    const evaluationRequest: EvaluationRequest = {
      flagKey,
      entityId,
      context
    };

    const result: BooleanResult | null = this.engine.evaluate_boolean(
      serialize(evaluationRequest)
    );

    if (result === null) {
      throw new Error('Failed to evaluate boolean');
    }

    const booleanResult = deserialize<BooleanResult>(result);

    if (booleanResult.status === 'failure') {
      throw new Error(booleanResult.errorMessage);
    }

    if (!booleanResult.result) {
      throw new Error('Failed to evaluate boolean');
    }

    return deserialize<BooleanEvaluationResponse>(booleanResult.result);
  }

  /**
   * Evaluate a batch of flag requests
   * @param requests evaluation requests
   * @returns {BatchEvaluationResponse}
   */
  public evaluateBatch(requests: EvaluationRequest[]): BatchEvaluationResponse {
    const serializedRequests = requests.map(serialize);
    const batchResult: BatchResult | null =
      this.engine.evaluate_batch(serializedRequests);

    if (batchResult === null) {
      throw new Error('Failed to evaluate batch');
    }

    if (batchResult.status === 'failure') {
      throw new Error(batchResult.errorMessage);
    }

    if (!batchResult.result) {
      throw new Error('Failed to evaluate batch');
    }

    const responses = batchResult.result.responses
      .map((response): EvaluationResponse | undefined => {
        if (response.type === 'BOOLEAN_EVALUATION_RESPONSE_TYPE') {
          const booleanResponse = deserialize<BooleanEvaluationResponse>(
            // @ts-ignore
            response.boolean_evaluation_response
          );
          return {
            booleanEvaluationResponse: booleanResponse,
            type: 'BOOLEAN_EVALUATION_RESPONSE_TYPE'
          };
        }
        if (response.type === 'VARIANT_EVALUATION_RESPONSE_TYPE') {
          const variantResponse = deserialize<VariantEvaluationResponse>(
            // @ts-ignore
            response.variant_evaluation_response
          );
          return {
            variantEvaluationResponse: variantResponse,
            type: 'VARIANT_EVALUATION_RESPONSE_TYPE'
          };
        }
        if (response.type === 'ERROR_EVALUATION_RESPONSE_TYPE') {
          const errorResponse = deserialize<ErrorEvaluationResponse>(
            // @ts-ignore
            response.error_evaluation_response
          );
          return {
            errorEvaluationResponse: errorResponse,
            type: 'ERROR_EVALUATION_RESPONSE_TYPE'
          };
        }
        return undefined;
      })
      .filter(
        (response): response is EvaluationResponse => response !== undefined
      );

    const requestDurationMillis = batchResult.result.requestDurationMillis;

    return {
      responses,
      requestDurationMillis
    };
  }

  public listFlags(): Flag[] {
    const listFlagsResult: ListFlagsResult | null = this.engine.list_flags();

    if (listFlagsResult === null) {
      throw new Error('Failed to list flags');
    }

    const flags = deserialize<ListFlagsResult>(listFlagsResult);

    if (flags.status === 'failure') {
      throw new Error(flags.errorMessage);
    }

    if (!flags.result) {
      throw new Error('Failed to list flags');
    }

    return flags.result.map(deserialize<Flag>);
  }

  public async close() {
    this.stopPollingUpdates();
    if (this.isStreaming) {
      this.stopStreamingUpdates();
      // Wait for the streaming to actually complete
      await new Promise<void>((resolve) => {
        const checkInterval = setInterval(() => {
          if (!this.isStreaming) {
            clearInterval(checkInterval);
            resolve();
          }
        }, 100);
      });
    }
  }
}

export * from './models';
