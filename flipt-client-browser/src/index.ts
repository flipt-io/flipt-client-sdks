import init, { Engine } from '../dist/flipt_engine_wasm.js';
import wasm from '../dist/flipt_engine_wasm_bg.wasm';

import {
  BatchEvaluationResponse,
  BatchResult,
  BooleanEvaluationResponse,
  BooleanResult,
  EngineOpts as Options,
  EvaluationRequest,
  IFetcher,
  VariantEvaluationResponse,
  VariantResult
} from './models.js';

export class FliptEvaluationClient {
  private engine: Engine;
  private fetcher: IFetcher;
  private etag?: string;

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
    options: Options = {
      url: 'http://localhost:8080',
      reference: ''
    }
  ): Promise<FliptEvaluationClient> {
    await init(await wasm());

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
    return new FliptEvaluationClient(engine, fetcher);
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
   * Evaluate a variant flag
   * @param flag_key - flag key to evaluate
   * @param entity_id - entity id to evaluate
   * @param context - optional evaluation context
   * @returns {VariantEvaluationResponse}
   */
  public evaluateVariant(
    flag_key: string,
    entity_id: string,
    context: {}
  ): VariantEvaluationResponse {
    const evaluation_request: EvaluationRequest = {
      flag_key,
      entity_id,
      context
    };

    const result = this.engine.evaluate_variant(
      evaluation_request
    ) as VariantResult;

    if (result.status === 'failure') {
      throw new Error(result.error_message);
    }

    return result.result as VariantEvaluationResponse;
  }

  /**
   * Evaluate a boolean flag
   * @param flag_key - flag key to evaluate
   * @param entity_id - entity id to evaluate
   * @param context - optional evaluation context
   * @returns {BooleanEvaluationResponse}
   */
  public evaluateBoolean(
    flag_key: string,
    entity_id: string,
    context: {}
  ): BooleanEvaluationResponse {
    const evaluation_request: EvaluationRequest = {
      flag_key,
      entity_id,
      context
    };

    const result = this.engine.evaluate_boolean(
      evaluation_request
    ) as BooleanResult;

    if (result.status === 'failure') {
      throw new Error(result.error_message);
    }

    return result.result as BooleanEvaluationResponse;
  }

  /**
   * Evaluate a batch of flag requests
   * @param requests evaluation requests
   * @returns {BatchEvaluationResponse}
   */
  public evaluateBatch(requests: EvaluationRequest[]): BatchEvaluationResponse {
    const result = this.engine.evaluate_batch(requests) as BatchResult;

    if (result.status === 'failure') {
      throw new Error(result.error_message);
    }

    return result.result as BatchEvaluationResponse;
  }
}
