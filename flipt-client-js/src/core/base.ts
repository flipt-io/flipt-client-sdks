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
  ListFlagsResult,
  Hook
} from './types';

import { deserialize, serialize } from './utils';

export type FliptClient = BaseFliptClient;

export abstract class BaseFliptClient {
  protected engine: any; // Type will be provided by platform implementations
  protected fetcher: IFetcher;
  protected etag?: string;
  protected errorStrategy?: ErrorStrategy;
  protected hook?: Hook;

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

    this.hook?.before({ flagKey });
    const variantResult: VariantResult | null = this.engine.evaluate_variant(
      serialize(evaluationRequest)
    );

    if (variantResult === null) {
      throw new Error('Failed to evaluate variant');
    }

    if (variantResult.status === 'failure') {
      throw new Error(variantResult.errorMessage);
    }

    if (!variantResult.result) {
      throw new Error('Failed to evaluate variant');
    }

    const result = deserialize<VariantEvaluationResponse>(variantResult.result);
    this.hook?.after({
      flagKey,
      flagType: 'variant',
      value: result.variantKey,
      reason: result.reason,
      segmentKeys: result.segmentKeys
    });
    return result;
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

    this.hook?.before({ flagKey });
    const booleanResult: BooleanResult | null = this.engine.evaluate_boolean(
      serialize(evaluationRequest)
    );

    if (booleanResult === null) {
      throw new Error('Failed to evaluate boolean');
    }

    if (booleanResult.status === 'failure') {
      throw new Error(booleanResult.errorMessage);
    }

    if (!booleanResult.result) {
      throw new Error('Failed to evaluate boolean');
    }

    const result = deserialize<BooleanEvaluationResponse>(booleanResult.result);
    this.hook?.after({
      flagKey,
      flagType: 'boolean',
      value: result.enabled.toString(),
      reason: result.reason,
      segmentKeys: result.segmentKeys
    });
    return result;
  }

  /**
   * Evaluate a batch of flag requests
   */
  public evaluateBatch(requests: EvaluationRequest[]): BatchEvaluationResponse {
    if (this.hook) {
      requests.forEach((req) => this.hook?.before({ flagKey: req.flagKey }));
    }
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
          this.hook?.after({
            flagKey: booleanResponse.flagKey,
            flagType: 'boolean',
            value: booleanResponse.enabled.toString(),
            reason: booleanResponse.reason,
            segmentKeys: booleanResponse.segmentKeys
          });
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
          this.hook?.after({
            flagKey: variantResponse.flagKey,
            flagType: 'variant',
            value: variantResponse.variantKey,
            reason: variantResponse.reason,
            segmentKeys: variantResponse.segmentKeys
          });
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

    if (listFlagsResult.status === 'failure') {
      throw new Error(listFlagsResult.errorMessage);
    }

    if (!listFlagsResult.result) {
      throw new Error('Failed to list flags');
    }

    return listFlagsResult.result.map(deserialize<Flag>);
  }
}
