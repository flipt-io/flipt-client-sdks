import init, { Engine } from '../dist/flipt_engine_wasm.js';
import wasm from '../dist/flipt_engine_wasm_bg.wasm';

import {
  BatchResult,
  BooleanResult,
  EngineOpts,
  EvaluationRequest,
  VariantResult
} from './models.js';

export class FliptEvaluationClient {
  private engine: Engine;
  private fetcher: () => Promise<Response>;

  private constructor(engine: Engine, fetcher: () => Promise<Response>) {
    this.engine = engine;
    this.fetcher = fetcher;
  }

  /**
   * Initialize the client
   * @param namespace - optional namespace to evaluate flags
   * @param engine_opts - optional engine options
   * @returns {Promise<FliptEvaluationClient>}
   */
  static async init(
    namespace: string = 'default',
    engine_opts: EngineOpts = {
      url: 'http://localhost:8080',
      reference: ''
    }
  ): Promise<FliptEvaluationClient> {
    await init(await wasm());

    let url = engine_opts.url ?? 'http://localhost:8080';
    // trim trailing slash
    url = url.replace(/\/$/, '');
    url = `${url}/internal/v1/evaluation/snapshot/namespace/${namespace}`;

    if (engine_opts.reference) {
      url = `${url}?ref=${engine_opts.reference}`;
    }

    const headers = new Headers();
    headers.append('Accept', 'application/json');
    headers.append('Content-Type', 'application/json');
    headers.append('x-flipt-accept-server-version', '1.38.0');

    if (engine_opts.authentication) {
      if ('client_token' in engine_opts.authentication) {
        headers.append(
          'Authorization',
          `Bearer ${engine_opts.authentication.client_token}`
        );
      } else if ('jwt_token' in engine_opts.authentication) {
        headers.append(
          'Authorization',
          `JWT ${engine_opts.authentication.jwt_token}`
        );
      }
    }

    let fetcher = engine_opts.fetcher;

    if (!fetcher) {
      fetcher = async () => {
        const resp = await fetch(url, {
          method: 'GET',
          headers
        });
        if (!resp.ok) {
          throw new Error('Failed to fetch data');
        }

        return resp;
      };
    }

    const resp = await fetcher();
    const data = await resp.json();

    const engine = new Engine(namespace, data);
    return new FliptEvaluationClient(engine, fetcher);
  }

  /**
   * Refresh the flags snapshot
   * @returns void
   */
  public async refresh() {
    const resp = await this.fetcher();
    const data = await resp.json();

    this.engine.snapshot(data);
  }

  /**
   * Evaluate a variant flag
   * @param flag_key - flag key to evaluate
   * @param entity_id - entity id to evaluate
   * @param context - optional evaluation context
   * @returns {VariantResult}
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

    return this.engine.evaluate_variant(evaluation_request) as VariantResult;
  }

  /**
   * Evaluate a boolean flag
   * @param flag_key - flag key to evaluate
   * @param entity_id - entity id to evaluate
   * @param context - optional evaluation context
   * @returns {BooleanResult}
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

    return this.engine.evaluate_boolean(evaluation_request) as BooleanResult;
  }

  /**
   * Evaluate a batch of flag requests
   * @param requests evaluation requests
   * @returns {BatchResult}
   */
  public evaluateBatch(requests: EvaluationRequest[]): BatchResult {
    return this.engine.evaluate_batch(requests);
  }
}
