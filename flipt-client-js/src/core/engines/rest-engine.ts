import { IEvaluationEngine } from './types';
import { RestClient } from './rest-client';

/**
 * REST-based Evaluation Engine
 *
 * Implements client-side flag evaluation using Flipt REST API.
 * This engine delegates evaluation to the Flipt server, making each evaluation
 * an asynchronous HTTP call to the evaluation endpoints.
 *
 * This is the CSP-compliant alternative to WASM evaluation:
 * - No WebAssembly required (no `wasm-unsafe-eval` CSP directive)
 * - Server-side evaluation (computation happens on Flipt server)
 * - Always up-to-date flag definitions (no snapshot caching)
 * - Higher latency (network round-trip per evaluation)
 *
 * Note: All methods return Promises and must be awaited.
 *
 * Responsibilities:
 * - Communicate with Flipt REST evaluation endpoints
 * - Serialize/deserialize evaluation requests and responses
 * - Handle errors from server responses
 * - Return evaluation results in standardized format
 *
 * @class RestEvaluationEngine
 * @implements {IEvaluationEngine}
 */
export class RestEvaluationEngine implements IEvaluationEngine {
  private client: RestClient;

  readonly isAsync = true;

  /**
   * Create a new REST evaluation engine
   *
   * @param baseUrl The base URL of the Flipt instance (e.g., 'http://localhost:8080')
   * @param namespace The namespace for flag evaluation
   * @param headers HTTP headers to include in all requests
   * @param fetcher The custom fetcher function (for testing or custom implementations)
   */
  constructor(
    baseUrl: string,
    namespace: string,
    headers: Record<string, string>
  ) {
    this.client = new RestClient(baseUrl, namespace, headers);
  }

  /**
   * Evaluate a variant flag (async)
   *
   * @override
   * @returns Promise that resolves to variant evaluation response
   */
  evaluateVariant(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): Promise<any> {
    return this._evaluateVariantAsync(request);
  }

  /**
   * Evaluate a variant flag asynchronously (internal implementation)
   *
   * @private
   */
  private async _evaluateVariantAsync(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): Promise<any> {
    const response = await this.client.evaluateVariant(request);
    return this._transformResponse(response);
  }

  /**
   * Evaluate a boolean flag (async)
   *
   * @override
   * @returns Promise that resolves to boolean evaluation response
   */
  evaluateBoolean(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): Promise<any> {
    return this._evaluateBooleanAsync(request);
  }

  /**
   * Evaluate a boolean flag asynchronously (internal implementation)
   *
   * @private
   */
  private async _evaluateBooleanAsync(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): Promise<any> {
    const response = await this.client.evaluateBoolean(request);
    return this._transformResponse(response);
  }

  /**
   * Evaluate a batch of flags (async)
   *
   * @override
   * @returns Promise that resolves to batch evaluation response
   */
  evaluateBatch(requests: any[]): Promise<any> {
    return this._evaluateBatchAsync(requests);
  }

  /**
   * Evaluate a batch of flags asynchronously (internal implementation)
   *
   * @private
   */
  private async _evaluateBatchAsync(
    requests: Array<{
      flagKey: string;
      entityId: string;
      context: Record<string, string>;
    }>
  ): Promise<any> {
    const response = await this.client.evaluateBatch(requests);

    // Transform batch response
    const responses = response.responses.map((resp: any) => {
      if (resp.variant_evaluation_response) {
        return {
          type: 'VARIANT_EVALUATION_RESPONSE_TYPE',
          variantEvaluationResponse: this._transformResponse(
            resp.variant_evaluation_response
          )
        };
      } else if (resp.boolean_evaluation_response) {
        return {
          type: 'BOOLEAN_EVALUATION_RESPONSE_TYPE',
          booleanEvaluationResponse: this._transformResponse(
            resp.boolean_evaluation_response
          )
        };
      } else if (resp.error_evaluation_response) {
        return {
          type: 'ERROR_EVALUATION_RESPONSE_TYPE',
          errorEvaluationResponse: resp.error_evaluation_response
        };
      }
      return null;
    });

    return {
      responses: responses.filter((r: any) => r !== null),
      requestDurationMillis: response.request_duration_millis || 0
    };
  }

  /**
   * List all available flags (async)
   *
   * @override
   * @returns Promise that resolves to array of flags
   */
  listFlags(): Promise<any[]> {
    return this._listFlagsAsync();
  }

  /**
   * List all available flags asynchronously (internal implementation)
   *
   * @private
   */
  private async _listFlagsAsync(): Promise<any[]> {
    const response = await this.client.listFlags();
    
    // Handle different response formats
    if (Array.isArray(response)) {
      return response;
    }
    const respObj = response as any;
    if (respObj && respObj.flags && Array.isArray(respObj.flags)) {
      return respObj.flags;
    }
    return [];
  }

  /**
   * Update the internal flag snapshot (no-op for REST engine)
   *
   * @override
   */
  snapshot(_data: any): void {
    // REST engine doesn't cache snapshots since it evaluates server-side
    // This is a no-op but required by IEvaluationEngine interface
  }

  /**
   * Transform snake_case response to camelCase
   *
   * @private
   */
  private _transformResponse(data: any): any {
    const result: any = {};
    for (const key in data) {
      if (Object.prototype.hasOwnProperty.call(data, key)) {
        const camelKey = this._snakeToCamel(key);
        result[camelKey] = data[key];
      }
    }
    return result;
  }

  /**
   * Convert snake_case to camelCase
   *
   * @private
   */
  private _snakeToCamel(str: string): string {
    return str.replace(/([-_][a-z])/g, (group) =>
      group.toUpperCase().replace('-', '').replace('_', '')
    );
  }
}
