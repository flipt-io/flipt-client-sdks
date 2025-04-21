import {
  BatchEvaluationResponse,
  BatchResult,
  BooleanEvaluationResponse,
  BooleanResult,
  ClientOptions,
  EvaluationRequest,
  ErrorStrategy,
  Flag,
  IFetcher,
  ListFlagsResult,
  Response,
  VariantResult,
  ErrorEvaluationResponse,
  EvaluationResponse,
  VariantEvaluationResponse
} from './types';
import { Engine } from '../wasm/flipt_engine_wasm_js.js';
import { deserialize, serialize } from './utils';

export interface InitOptions {
  options: ClientOptions;
  initWasm: () => Promise<any>;
  createClient: (engine: Engine, fetcher: IFetcher) => BaseClient;
}

export abstract class BaseClient {
  protected engine: Engine;
  protected fetcher: IFetcher;
  protected etag?: string;
  protected errorStrategy: ErrorStrategy = ErrorStrategy.Fail;
  protected lastGoodState?: any;

  constructor(engine: Engine, fetcher: IFetcher) {
    this.engine = engine;
    this.fetcher = fetcher;
  }

  protected static async initialize({
    options,
    initWasm,
    createClient
  }: InitOptions): Promise<BaseClient> {
    const namespace = options.namespace ?? 'default';

    let url = options.url ?? 'http://localhost:8080';
    url = url.replace(/\/$/, '');
    url = `${url}/internal/v1/evaluation/snapshot/namespace/${namespace}`;

    if (options.reference) {
      url = `${url}?reference=${options.reference}`;
    }

    const headers: Record<string, string> = {
      Accept: 'application/json',
      'x-flipt-accept-server-version': '1.47.0'
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
      throw new Error('No fetcher provided and no default fetcher available');
    }

    // Initialize WASM engine
    await initWasm();

    // handle case if they pass in a custom fetcher that doesn't throw on non-2xx status codes
    const resp = await fetcher();
    if (!resp.ok) {
      throw new Error(`Failed to fetch data: ${resp.statusText}`);
    }

    const data = await resp.json();
    const engine = new Engine(namespace);
    engine.snapshot(data);

    const client = createClient(engine, fetcher);

    client.storeEtag(resp);
    client.errorStrategy = options.errorStrategy ?? ErrorStrategy.Fail;

    return client;
  }

  protected storeEtag(resp: Response): void {
    const etag = resp.headers.get('ETag');
    if (etag) {
      this.etag = etag;
    }
  }

  /**
   * Evaluate a variant flag
   */
  public evaluateVariant({
    flagKey,
    entityId,
    context
  }: {
    flagKey: string;
    entityId: string;
    context: Record<string, unknown>;
  }): VariantEvaluationResponse {
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
   */
  public evaluateBoolean({
    flagKey,
    entityId,
    context
  }: {
    flagKey: string;
    entityId: string;
    context: Record<string, unknown>;
  }): BooleanEvaluationResponse {
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

    return {
      responses,
      requestDurationMillis: batchResult.result.requestDurationMillis
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

  /**
   * Refresh the flags snapshot
   * @returns true if snapshot changed
   */
  public async refresh(): Promise<boolean> {
    try {
      const opts = { etag: this.etag };
      const resp = await this.fetcher(opts);

      const etag = resp.headers.get('etag');
      if (this.etag && this.etag === etag) {
        return false;
      }

      this.storeEtag(resp);

      const data = await resp.json();
      this.engine.snapshot(data);
      return true;
    } catch (error) {
      if (this.errorStrategy === ErrorStrategy.Fail) {
        throw error;
      }
      return false;
    }
  }
}
