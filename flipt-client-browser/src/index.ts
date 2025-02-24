import init, { Engine } from '../dist/flipt_engine_wasm.js';
import wasm from '../dist/flipt_engine_wasm_bg.wasm';
import { deserialize, serialize } from './utils';
import {
  BatchEvaluationResponse,
  BooleanEvaluationResponse,
  ClientOptions,
  ErrorEvaluationResponse,
  EvaluationRequest,
  EvaluationResponse,
  Flag,
  IFetcher,
  VariantEvaluationResponse,
  ErrorStrategy
} from './models';

import {
  VariantResult,
  BooleanResult,
  BatchResult,
  ListFlagsResult
} from './internal/models';

export class FliptEvaluationClient {
  private engine: Engine;
  private fetcher: IFetcher;
  private etag?: string;
  private errorStrategy?: ErrorStrategy;

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
      reference: '',
      errorStrategy: ErrorStrategy.Fail
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

        if (!resp.ok && resp.status !== 304) {
          throw new Error(`Failed to fetch data: ${resp.statusText}`);
        }

        return resp;
      };
    }

    // handle case if they pass in a custom fetcher that doesn't throw on non-2xx status codes
    const resp = await fetcher();
    if (!resp.ok) {
      throw new Error(`Failed to fetch data: ${resp.statusText}`);
    }

    const data = await resp.json();
    const engine = new Engine(namespace);
    engine.snapshot(data);
    const client = new FliptEvaluationClient(engine, fetcher);
    client.storeEtag(resp);
    client.errorStrategy = options.errorStrategy;
    return client;
  }

  /**
   * Store etag from response for next requests
   */
  private storeEtag(resp: Response) {
    let etag = resp.headers.get('etag');
    if (etag) {
      this.etag = etag;
    }
  }

  /**
   * Refresh the flags snapshot
   * @returns true if snapshot changed
   */
  public async refresh(): Promise<boolean> {
    try {
      const opts = { etag: this.etag };
      const resp = await this.fetcher(opts);

      let etag = resp.headers.get('etag');
      if (this.etag && this.etag === etag) {
        return false;
      }

      this.storeEtag(resp);

      const data = await resp.json();
      this.engine.snapshot(data);
      return true;
    } catch (error) {
      if (this.errorStrategy === ErrorStrategy.Fail) {
        throw error; // Re-throw the error
      }
    }
    return false;
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
    if (!flagKey || flagKey.trim() === '') {
      throw new Error('flagKey cannot be empty');
    }
    if (!entityId || entityId.trim() === '') {
      throw new Error('entityId cannot be empty');
    }

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
    if (!flagKey || flagKey.trim() === '') {
      throw new Error('flagKey cannot be empty');
    }
    if (!entityId || entityId.trim() === '') {
      throw new Error('entityId cannot be empty');
    }

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
}

export * from './models';
