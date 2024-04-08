import init, { Engine } from '../dist/flipt_engine_wasm.js';
import { BooleanResult, EngineOpts, VariantResult } from './models.js';

interface IEvaluationRequest {
  flag_key: string;
  entity_id: string;
  context: object;
}

export class FliptEvaluationClient {
  private engine: Engine;
  private fetcher: () => Promise<Response>;

  private constructor(engine: Engine, fetcher: () => Promise<Response>) {
    this.engine = engine;
    this.fetcher = fetcher;
  }

  static async init(
    namespace: string = 'default',
    engine_opts: EngineOpts = {
      url: 'http://localhost:8080',
      reference: ''
    }
  ) {
    await init();
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

    const fetcher = async () => {
      const resp = await fetch(url, {
        method: 'GET',
        headers
      });

      if (!resp.ok) {
        throw new Error('Failed to fetch data');
      }

      return resp;
    };

    const resp = await fetcher();
    const data = await resp.json();

    const engine = new Engine(namespace, data);
    return new FliptEvaluationClient(engine, fetcher);
  }

  public async refresh() {
    const resp = await this.fetcher();
    const data = await resp.json();

    this.engine.snapshot(data);
  }

  public evaluateVariant(flag_key: string, entity_id: string, context: {}) {
    const evaluation_request: IEvaluationRequest = {
      flag_key,
      entity_id,
      context
    };

    return this.engine.evaluate_variant(evaluation_request) as VariantResult;
  }

  public evaluateBoolean(flag_key: string, entity_id: string, context: {}) {
    const evaluation_request: IEvaluationRequest = {
      flag_key,
      entity_id,
      context
    };

    return this.engine.evaluate_boolean(evaluation_request) as BooleanResult;
  }
}
