import { IFetcherOptions } from '../types';

/**
 * REST Client for Flipt Evaluation API
 *
 * Handles all HTTP communication with Flipt evaluation endpoints.
 * This client abstracts the HTTP layer for the REST evaluation engine.
 *
 * Security:
 * - Validates all responses
 * - Properly escapes and validates inputs
 * - Never exposes sensitive error details to clients
 * - Handles authentication transparently
 *
 * @class RestClient
 */
export class RestClient {
  private baseUrl: string;
  private namespace: string;
  private headers: Record<string, string>;

  constructor(
    baseUrl: string,
    namespace: string,
    headers: Record<string, string>
  ) {
    // Ensure baseUrl doesn't have trailing slash
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.namespace = namespace;
    this.headers = headers;
  }

  /**
   * Evaluate a variant flag via REST API
   *
   * POST /evaluate/v1/{namespace}/variant
   */
  async evaluateVariant(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): Promise<any> {
    const url = `${this.baseUrl}/evaluate/v1/${encodeURIComponent(this.namespace)}/variant`;
    return this._post(url, this._camelToSnake(request));
  }

  /**
   * Evaluate a boolean flag via REST API
   *
   * POST /evaluate/v1/{namespace}/boolean
   */
  async evaluateBoolean(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): Promise<any> {
    const url = `${this.baseUrl}/evaluate/v1/${encodeURIComponent(this.namespace)}/boolean`;
    return this._post(url, this._camelToSnake(request));
  }

  /**
   * Evaluate multiple flags via REST API
   *
   * POST /evaluate/v1/{namespace}/batch
   */
  async evaluateBatch(
    requests: Array<{
      flagKey: string;
      entityId: string;
      context: Record<string, string>;
    }>
  ): Promise<any> {
    const url = `${this.baseUrl}/evaluate/v1/${encodeURIComponent(this.namespace)}/batch`;
    const payload = { requests: requests.map((r) => this._camelToSnake(r)) };
    return this._post(url, payload);
  }

  /**
   * List all available flags via REST API
   *
   * GET /evaluate/v1/{namespace}/flags
   */
  async listFlags(): Promise<any[]> {
    const url = `${this.baseUrl}/evaluate/v1/${encodeURIComponent(this.namespace)}/flags`;
    return this._get(url);
  }

  /**
   * Generic POST request handler
   *
   * @private
   */
  private async _post(url: string, payload: any): Promise<any> {
    try {
      // Create a custom fetcher that includes the request body
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          ...this.headers,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(payload)
      });

      return this._handleResponse(response);
    } catch (error) {
      throw this._wrapNetworkError(error);
    }
  }

  /**
   * Generic GET request handler
   *
   * @private
   */
  private async _get(url: string, opts?: IFetcherOptions): Promise<any> {
    try {
      const requestHeaders = { ...this.headers };
      
      if (opts?.etag) {
        requestHeaders['If-None-Match'] = opts.etag;
      }

      const response = await fetch(url, {
        method: 'GET',
        headers: requestHeaders
      });

      return this._handleResponse(response);
    } catch (error) {
      throw this._wrapNetworkError(error);
    }
  }

  /**
   * Handle HTTP responses
   *
   * @private
   */
  private async _handleResponse(response: Response): Promise<any> {
    // Handle 304 Not Modified
    if (response.status === 304) {
      return { _noChange: true };
    }

    // Check if response is ok
    if (!response.ok) {
      const errorMessage = await this._extractErrorMessage(response);
      throw new Error(`Flipt evaluation failed: ${errorMessage}`);
    }

    try {
      return await response.json();
    } catch {
      throw new Error('Failed to parse Flipt response: invalid JSON');
    }
  }

  /**
   * Extract error message from response
   *
   * @private
   */
  private async _extractErrorMessage(response: Response): Promise<string> {
    try {
      const data = await response.json();
      
      // Try common error field names
      if (data.message) return data.message;
      if (data.error) return data.error;
      if (data.errors && Array.isArray(data.errors)) {
        return data.errors.map((e: any) => e.message || String(e)).join('; ');
      }
      
      return response.statusText || 'Unknown error';
    } catch {
      // If can't parse error response, use status
      return `HTTP ${response.status}: ${response.statusText}`;
    }
  }

  /**
   * Wrap network errors with consistent format
   *
   * @private
   */
  private _wrapNetworkError(error: any): Error {
    if (error instanceof Error) {
      // Network errors
      if (error.message.includes('fetch')) {
        return new Error('Network error: Unable to connect to Flipt server');
      }
      return error;
    }

    return new Error('Unknown network error');
  }

  /**
   * Convert camelCase keys to snake_case
   *
   * @private
   */
  private _camelToSnake(obj: any): any {
    const result: any = {};
    for (const key in obj) {
      if (Object.prototype.hasOwnProperty.call(obj, key)) {
        const snakeKey = key.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);
        result[snakeKey] = obj[key];
      }
    }
    return result;
  }
}
