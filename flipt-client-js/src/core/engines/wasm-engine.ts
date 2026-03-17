import { IEvaluationEngine, EngineResult } from './types';

/**
 * WASM-based Evaluation Engine
 *
 * Implements client-side flag evaluation using WebAssembly.
 * This is the default, high-performance evaluation strategy.
 *
 * Responsibilities:
 * - Load and initialize WASM engine
 * - Perform in-process flag evaluation (requires wasm-unsafe-eval CSP)
 * - Serialize/deserialize flag data
 * - Return evaluation results in standardized format
 *
 * @class WasmEngine
 * @implements {IEvaluationEngine}
 */
export class WasmEngine implements IEvaluationEngine {
  private engine: any; // WASM Engine instance

  /**
   * Create a new WASM evaluation engine
   *
   * @param wasmEngine The initialized WASM engine instance
   * @param namespace The namespace for flag evaluation
   */
  constructor(wasmEngine: any, private namespace: string) {
    this.engine = wasmEngine;
  }

  /**
   * Evaluate a variant flag
   *
   * @override
   */
  evaluateVariant(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): any {
    const evaluationRequest = {
      flag_key: request.flagKey,
      entity_id: request.entityId,
      context: request.context
    };

    const variantResult: EngineResult<any> | null = this.engine.evaluate_variant(
      evaluationRequest
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

    return this._deserialize(variantResult.result);
  }

  /**
   * Evaluate a boolean flag
   *
   * @override
   */
  evaluateBoolean(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): any {
    const evaluationRequest = {
      flag_key: request.flagKey,
      entity_id: request.entityId,
      context: request.context
    };

    const booleanResult: EngineResult<any> | null = this.engine.evaluate_boolean(
      evaluationRequest
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

    return this._deserialize(booleanResult.result);
  }

  /**
   * Evaluate a batch of flags
   *
   * @override
   */
  evaluateBatch(
    requests: Array<{
      flagKey: string;
      entityId: string;
      context: Record<string, string>;
    }>
  ): any {
    const serializedRequests = requests.map((req) => ({
      flag_key: req.flagKey,
      entity_id: req.entityId,
      context: req.context
    }));

    const batchResult: EngineResult<any> | null = this.engine.evaluate_batch(
      serializedRequests
    );

    if (batchResult === null) {
      throw new Error('Failed to evaluate batch');
    }

    if (batchResult.status === 'failure') {
      throw new Error(batchResult.errorMessage);
    }

    if (!batchResult.result) {
      throw new Error('Failed to evaluate batch');
    }

    const responses = batchResult.result.responses.map(
      (response: any): any => {
        if (response.type === 'BOOLEAN_EVALUATION_RESPONSE_TYPE') {
          return {
            booleanEvaluationResponse: this._deserialize(
              response.boolean_evaluation_response
            ),
            type: 'BOOLEAN_EVALUATION_RESPONSE_TYPE'
          };
        }
        if (response.type === 'VARIANT_EVALUATION_RESPONSE_TYPE') {
          return {
            variantEvaluationResponse: this._deserialize(
              response.variant_evaluation_response
            ),
            type: 'VARIANT_EVALUATION_RESPONSE_TYPE'
          };
        }
        if (response.type === 'ERROR_EVALUATION_RESPONSE_TYPE') {
          return {
            errorEvaluationResponse: this._deserialize(
              response.error_evaluation_response
            ),
            type: 'ERROR_EVALUATION_RESPONSE_TYPE'
          };
        }
        return undefined;
      }
    );

    return {
      responses: responses.filter((r: any) => r !== undefined),
      requestDurationMillis: batchResult.result.request_duration_millis
    };
  }

  /**
   * List all available flags
   *
   * @override
   */
  listFlags(): any[] {
    const listFlagsResult: EngineResult<any> | null = this.engine.list_flags();

    if (listFlagsResult === null) {
      throw new Error('Failed to list flags');
    }

    if (listFlagsResult.status === 'failure') {
      throw new Error(listFlagsResult.errorMessage);
    }

    if (!listFlagsResult.result) {
      throw new Error('Failed to list flags');
    }

    return listFlagsResult.result.map((flag: any) => this._deserialize(flag));
  }

  /**
   * Update the internal flag snapshot
   *
   * @override
   */
  snapshot(data: any): void {
    this.engine.snapshot(data);
  }

  /**
   * Convert snake_case keys to camelCase
   * Used for deserializing WASM engine responses
   *
   * @private
   */
  private _snakeToCamel(str: string): string {
    return str.replace(/([-_][a-z])/g, (group) =>
      group.toUpperCase().replace('-', '').replace('_', '')
    );
  }

  /**
   * Deserialize WASM engine response (snake_case → camelCase)
   *
   * @private
   */
  private _deserialize(data: any): any {
    const result: any = {};
    for (const key in data) {
      if (Object.prototype.hasOwnProperty.call(data, key)) {
        const camelKey = this._snakeToCamel(key);
        result[camelKey] = data[key];
      }
    }
    return result;
  }
}
