import {
  BatchEvaluationResponse,
  BooleanEvaluationResponse,
  EvaluationRequest,
  EvaluationResponse,
  Flag,
  IFetcher,
  VariantEvaluationResponse,
  ErrorStrategy,
  Response,
  Hook
} from './types';

import { IEvaluationEngine } from './engines/types';

export type FliptClient = BaseFliptClient;

export abstract class BaseFliptClient {
  protected engine: IEvaluationEngine;
  protected fetcher: IFetcher;
  protected etag?: string;
  protected errorStrategy?: ErrorStrategy;
  protected hook?: Hook;

  constructor(engine: IEvaluationEngine, fetcher: IFetcher) {
    this.engine = engine;
    this.fetcher = fetcher;
  }

  private assertSyncEngine(methodName: string) {
    if (this.engine.isAsync) {
      throw new Error(
        `${methodName} is synchronous and not available in REST mode. Use ${methodName}Async instead.`
      );
    }
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
    this.assertSyncEngine('evaluateVariant');

    if (!flagKey || flagKey.trim() === '') {
      throw new Error('flagKey cannot be empty');
    }
    if (!entityId || entityId.trim() === '') {
      throw new Error('entityId cannot be empty');
    }

    this.hook?.before({ flagKey });

    const result = this.engine.evaluateVariant({
      flagKey,
      entityId,
      context
    }) as VariantEvaluationResponse;

    this.hook?.after({
      flagKey,
      flagType: 'variant',
      value: result.variantKey,
      reason: result.reason,
      segmentKeys: result.segmentKeys
    });

    return result;
  }

  public async evaluateVariantAsync({
    flagKey,
    entityId,
    context
  }: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): Promise<VariantEvaluationResponse> {
    if (!flagKey || flagKey.trim() === '') {
      throw new Error('flagKey cannot be empty');
    }
    if (!entityId || entityId.trim() === '') {
      throw new Error('entityId cannot be empty');
    }

    this.hook?.before({ flagKey });

    const result = (await Promise.resolve(
      this.engine.evaluateVariant({
        flagKey,
        entityId,
        context
      })
    )) as VariantEvaluationResponse;

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
    this.assertSyncEngine('evaluateBoolean');

    if (!flagKey || flagKey.trim() === '') {
      throw new Error('flagKey cannot be empty');
    }
    if (!entityId || entityId.trim() === '') {
      throw new Error('entityId cannot be empty');
    }

    this.hook?.before({ flagKey });

    const result = this.engine.evaluateBoolean({
      flagKey,
      entityId,
      context
    }) as BooleanEvaluationResponse;

    this.hook?.after({
      flagKey,
      flagType: 'boolean',
      value: result.enabled.toString(),
      reason: result.reason,
      segmentKeys: result.segmentKeys
    });

    return result;
  }

  public async evaluateBooleanAsync({
    flagKey,
    entityId,
    context
  }: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): Promise<BooleanEvaluationResponse> {
    if (!flagKey || flagKey.trim() === '') {
      throw new Error('flagKey cannot be empty');
    }
    if (!entityId || entityId.trim() === '') {
      throw new Error('entityId cannot be empty');
    }

    this.hook?.before({ flagKey });
    const result = (await Promise.resolve(
      this.engine.evaluateBoolean({
        flagKey,
        entityId,
        context
      })
    )) as BooleanEvaluationResponse;

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
    this.assertSyncEngine('evaluateBatch');

    if (this.hook) {
      requests.forEach((req) => this.hook?.before({ flagKey: req.flagKey }));
    }

    const batchResult = this.engine.evaluateBatch(
      requests.map((req) => ({
        flagKey: req.flagKey,
        entityId: req.entityId,
        context: req.context
      }))
    ) as BatchEvaluationResponse;

    // Fire after hooks for each response
    batchResult.responses.forEach((response: EvaluationResponse) => {
      if (response.type === 'BOOLEAN_EVALUATION_RESPONSE_TYPE') {
        const booleanResponse = response.booleanEvaluationResponse;
        if (!booleanResponse) {
          return;
        }
        this.hook?.after({
          flagKey: booleanResponse.flagKey,
          flagType: 'boolean',
          value: booleanResponse.enabled.toString(),
          reason: booleanResponse.reason,
          segmentKeys: booleanResponse.segmentKeys
        });
      } else if (response.type === 'VARIANT_EVALUATION_RESPONSE_TYPE') {
        const variantResponse = response.variantEvaluationResponse;
        if (!variantResponse) {
          return;
        }
        this.hook?.after({
          flagKey: variantResponse.flagKey,
          flagType: 'variant',
          value: variantResponse.variantKey,
          reason: variantResponse.reason,
          segmentKeys: variantResponse.segmentKeys
        });
      }
    });

    return batchResult;
  }

  public async evaluateBatchAsync(
    requests: EvaluationRequest[]
  ): Promise<BatchEvaluationResponse> {
    if (this.hook) {
      requests.forEach((req) => this.hook?.before({ flagKey: req.flagKey }));
    }

    const batchResult = (await Promise.resolve(
      this.engine.evaluateBatch(
        requests.map((req) => ({
          flagKey: req.flagKey,
          entityId: req.entityId,
          context: req.context
        }))
      )
    )) as BatchEvaluationResponse;

    batchResult.responses.forEach((response: EvaluationResponse) => {
      if (response.type === 'BOOLEAN_EVALUATION_RESPONSE_TYPE') {
        const booleanResponse = response.booleanEvaluationResponse;
        if (!booleanResponse) {
          return;
        }
        this.hook?.after({
          flagKey: booleanResponse.flagKey,
          flagType: 'boolean',
          value: booleanResponse.enabled.toString(),
          reason: booleanResponse.reason,
          segmentKeys: booleanResponse.segmentKeys
        });
      } else if (response.type === 'VARIANT_EVALUATION_RESPONSE_TYPE') {
        const variantResponse = response.variantEvaluationResponse;
        if (!variantResponse) {
          return;
        }
        this.hook?.after({
          flagKey: variantResponse.flagKey,
          flagType: 'variant',
          value: variantResponse.variantKey,
          reason: variantResponse.reason,
          segmentKeys: variantResponse.segmentKeys
        });
      }
    });

    return batchResult;
  }

  public listFlags(): Flag[] {
    this.assertSyncEngine('listFlags');
    return this.engine.listFlags() as Flag[];
  }

  public async listFlagsAsync(): Promise<Flag[]> {
    return (await Promise.resolve(this.engine.listFlags())) as Flag[];
  }
}
