import {
  BatchEvaluationResponse,
  BooleanEvaluationResponse,
  ErrorEvaluationResponse,
  EvaluationRequest,
  EvaluationResponse,
  Flag,
  IFetcher,
  VariantEvaluationResponse,
  ErrorStrategy,
  Response,
  VariantResult,
  BooleanResult,
  BatchResult,
  ListFlagsResult
} from './types';

import { deserialize, serialize } from './utils';

export type FliptClient = BaseFliptClient;

export abstract class BaseFliptClient {
  protected engine: any; // Type will be provided by platform implementations
  protected fetcher: IFetcher;
  protected etag?: string;
  protected errorStrategy?: ErrorStrategy;

  constructor(engine: any, fetcher: IFetcher) {
    this.engine = engine;
    this.fetcher = fetcher;
  }

  /**
   * Store etag from response for next requests
   */
  protected storeEtag(resp: Response) {
    const etag = resp.headers.get('etag');
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
    }
    return false;
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
    context: Record<string, string>;
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
    context: Record<string, string>;
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
}
