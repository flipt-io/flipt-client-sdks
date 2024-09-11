import { Engine } from '../dist/flipt_engine_wasm.js';
import { serialize, deserialize } from './utils';

import {
  AuthenticationStrategy,
  BooleanResult,
  BatchResult,
  ClientOptions,
  EvaluationRequest,
  IFetcher,
  VariantResult,
  VariantEvaluationResponse,
  BooleanEvaluationResponse,
  BatchEvaluationResponse,
  ErrorEvaluationResponse,
  EvaluationResponse
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

  static async init(
    namespace: string = 'default',
    options: ClientOptions<AuthenticationStrategy> = {
      url: 'http://localhost:8080',
      reference: '',
      updateInterval: 120
    }
  ): Promise<FliptEvaluationClient> {
    let url = options.url ?? 'http://localhost:8080';
    url = url.replace(/\/$/, '');
    url = `${url}/internal/v1/evaluation/snapshot/namespace/${namespace}`;

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
      fetcher = async (opts?: { etag?: string }) => {
        if (opts && opts.etag) {
          headers.append('If-None-Match', opts.etag);
        }

        const resp = await fetch(url, {
          method: 'GET',
          headers
        });

        if (resp.status === 304) {
          return resp;
        }

        if (!resp.ok) {
          throw new Error('Failed to fetch data');
        }

        return resp;
      };
    }

    const resp = await fetcher();
    if (!resp) {
      throw new Error('Failed to fetch data');
    }

    const data = await resp.json();
    const engine = new Engine(namespace, data);
    const client = new FliptEvaluationClient(engine, fetcher);

    if (options.updateInterval && options.updateInterval > 0) {
      client.startAutoRefresh(options.updateInterval * 1000);
    }

    return client;
  }

  private startAutoRefresh(interval: number = 120_000) {
    this.stopAutoRefresh();
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
          // @ts-ignore
          const booleanResponse = deserialize<BooleanEvaluationResponse>(response.boolean_evaluation_response);
          return {
            booleanEvaluationResponse: booleanResponse,
            type: 'BOOLEAN_EVALUATION_RESPONSE_TYPE'
          };
        }
        if (response.type === 'VARIANT_EVALUATION_RESPONSE_TYPE') {
          // @ts-ignore
          const variantResponse = deserialize<VariantEvaluationResponse>(response.variant_evaluation_response);
          return {
            variantEvaluationResponse: variantResponse,
            type: 'VARIANT_EVALUATION_RESPONSE_TYPE'
          };
        }
        if (response.type === 'ERROR_EVALUATION_RESPONSE_TYPE') {
          // @ts-ignore
          const errorResponse = deserialize<ErrorEvaluationResponse>(response.error_evaluation_response);
          return {
            errorEvaluationResponse: errorResponse,
            type: 'ERROR_EVALUATION_RESPONSE_TYPE'
          };
        }
        return undefined;
      })
      .filter((response): response is EvaluationResponse => response !== undefined);

    const requestDurationMillis = batchResult.result.requestDurationMillis;

    return {
      responses,
      requestDurationMillis
    };
  }

  public close() {
    this.stopAutoRefresh();
  }
}
