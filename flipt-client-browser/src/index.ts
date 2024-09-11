import init, { Engine } from '../dist/flipt_engine_wasm.js';
import wasm from '../dist/flipt_engine_wasm_bg.wasm';
import { deserialize, serialize } from './utils';
import {
  BatchEvaluationResponse,
  BatchResult,
  BooleanEvaluationResponse,
  BooleanResult,
  ClientOptions,
  EvaluationRequest,
  IFetcher,
  VariantEvaluationResponse,
  VariantResult
} from './models';

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
    options: ClientOptions = {
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

    const result = this.engine.evaluate_variant(serialize(evaluationRequest));

    const variantResult = deserialize<VariantResult>(result);

    if (variantResult.status === 'failure') {
      throw new Error(result.errorMessage);
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

    const result = this.engine.evaluate_boolean(serialize(evaluationRequest));

    const booleanResult = deserialize<BooleanResult>(result);

    if (booleanResult.status === 'failure') {
      throw new Error(booleanResult.errorMessage);
    }

    return deserialize<BooleanEvaluationResponse>(booleanResult.result);
  }

  /**
   * Evaluate a batch of flag requests
   * @param requests evaluation requests
   * @returns {BatchEvaluationResponse}
   */
  public evaluateBatch(requests: EvaluationRequest[]): BatchEvaluationResponse {
    const result = this.engine.evaluate_batch(serialize(requests));

    const batchResult = deserialize<BatchResult>(result);

    if (batchResult.status === 'failure') {
      throw new Error(batchResult.errorMessage);
    }

    return deserialize<BatchEvaluationResponse>(batchResult.result);
  }
}
