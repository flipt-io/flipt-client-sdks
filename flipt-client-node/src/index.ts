import { Engine } from '../dist/flipt_engine_wasm.js';

import {
  AuthenticationStrategy,
  BooleanResult,
  BatchResult,
  ClientOptions,
  EvaluationRequest,
  IFetcher,
  VariantResult
} from './models';

export class FliptEvaluationClient {
  private engine: Engine;
  private fetcher: IFetcher;
  private etag?: string;
  private updateInterval?: NodeJS.Timeout;

  private constructor(engine: Engine, fetcher: IFetcher) {
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
      update_interval: 120000
    }
  ): Promise<FliptEvaluationClient> {
    let url = options.url ?? 'http://localhost:8080';
    // trim trailing slash
    url = url.replace(/\/$/, '');
    url = `${url}/internal/v1/evaluation/snapshot/namespace/${namespace}`;

    if (options.reference) {
      url = `${url}?reference=${options.reference}`;
    }

    const headers = new Headers();
    headers.append('Accept', 'application/json');
    headers.append('x-flipt-accept-server-version', '1.47.0');

    if (options.authentication) {
      if ('client_token' in options.authentication) {
        headers.append(
          'Authorization',
          `Bearer ${options.authentication.client_token}`
        );
      } else if ('jwt_token' in options.authentication) {
        headers.append(
          'Authorization',
          `JWT ${options.authentication.jwt_token}`
        );
      }
    }

    let fetcher = options.fetcher;

    if (!fetcher) {
      fetcher = async (opts?: { etag?: string }) => {
        if (opts && opts.etag) {
          headers.append('If-None-Match', opts.etag);
        }

        const resp = await fetch(url, {
          method: 'GET',
          headers
        });

        // check for 304 status code
        if (resp.status === 304) {
          return resp;
        }

        // ok only checks for range 200-299
        if (!resp.ok) {
          throw new Error('Failed to fetch data');
        }

        return resp;
      };
    }

    // should be no etag on first fetch
    const resp = await fetcher();
    if (!resp) {
      throw new Error('Failed to fetch data');
    }

    const data = await resp.json();
    const engine = new Engine(namespace, data);
    const client = new FliptEvaluationClient(engine, fetcher);

    if (options.update_interval && options.update_interval > 0) {
      client.startAutoRefresh(options.update_interval);
    }

    return client;
  }

  private startAutoRefresh(interval: number = 120000) {
    this.stopAutoRefresh(); // Ensure we don't have multiple intervals running
    this.updateInterval = setInterval(async () => {
      await this.refresh();
    }, interval);
  }

  private stopAutoRefresh() {
    if (this.updateInterval) {
      clearInterval(this.updateInterval);
      this.updateInterval = undefined;
    }
  }

  /**
   * Refresh the flags snapshot
   * @returns void
   */
  public async refresh() {
    const opts = { etag: this.etag };
    const resp = await this.fetcher(opts);

    if (resp.status === 304) {
      let etag = resp.headers.get('etag');
      if (etag) {
        this.etag = etag;
      }
      return;
    }

    const data = await resp.json();
    this.engine.snapshot(data);
  }

  /**
   * Performs evaluation for a variant flag.
   *
   * @param flag_key - Feature flag key. {@link EvaluationRequest.flag_key}
   * @param entity_id - Entity identifier. {@link EvaluationRequest.entity_id}
   * @param context - Context information for flag evaluation.  {@link EvaluationRequest.context}
   *
   * @returns {@link VariantResult} the result of evaluation
   *
   * @throws {@link Error}
   * This exception is thrown if unexpected error occurs.
   */
  public evaluateVariant(
    flag_key: string,
    entity_id: string,
    context: {}
  ): VariantResult {
    const evaluation_request: EvaluationRequest = {
      flag_key,
      entity_id,
      context
    };

    const response = this.engine.evaluate_variant(evaluation_request);

    if (response === null) {
      throw new Error('Failed to evaluate variant');
    }

    return response;
  }

  /**
   * Performs evaluation for a boolean flag.
   *
   * @param flag_key - Feature flag key. {@link EvaluationRequest.flag_key}
   * @param entity_id - Entity identifier. {@link EvaluationRequest.entity_id}
   * @param context - Context information for flag evaluation.  {@link EvaluationRequest.context}
   *
   * @returns {@link BooleanResult} the result of evaluation
   *
   * @throws {@link Error}
   * This exception is thrown if unexpected error occurs.
   */
  public evaluateBoolean(
    flag_key: string,
    entity_id: string,
    context: {}
  ): BooleanResult {
    const evaluation_request: EvaluationRequest = {
      flag_key,
      entity_id,
      context
    };

    const response = this.engine.evaluate_boolean(evaluation_request);

    if (response === null) {
      throw new Error('Failed to evaluate boolean');
    }

    return response;
  }

  /**
   * Performs batch evaluation for bulk of flags.
   *
   * @param requests - array of evaluation requests {@link EvaluationRequest}.
   *
   * @returns {@link BatchResult}
   *
   * @throws {@link Error}
   * This exception is thrown if unexpected error occurs.
   */
  public evaluateBatch(requests: EvaluationRequest[]): BatchResult {
    const response = this.engine.evaluate_batch(requests);

    if (response === null) {
      throw new Error('Failed to evaluate batch');
    }

    return response;
  }

  public close() {
    this.stopAutoRefresh();
  }
}
