/**
 * Evaluation Engine Interface
 *
 * Defines the contract that all evaluation engines must implement.
 * This abstraction enables support for multiple evaluation strategies:
 * - WASM-based (local, fast evaluation)
 * - JS-Local (pure JavaScript, CSP-compliant client-side evaluation)
 * - REST-based (server-side evaluation for CSP compliance)
 *
 * @interface IEvaluationEngine
 */
export interface IEvaluationEngine {
  /**
   * Evaluate a variant flag
   *
   * @param request The evaluation request containing flagKey, entityId, and context
   * @returns The variant evaluation response (or Promise if async engine)
   * @throws Error if evaluation fails
   */
  evaluateVariant(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): any;

  /**
   * Evaluate a boolean flag
   *
   * @param request The evaluation request containing flagKey, entityId, and context
   * @returns The boolean evaluation response (or Promise if async engine)
   * @throws Error if evaluation fails
   */
  evaluateBoolean(request: {
    flagKey: string;
    entityId: string;
    context: Record<string, string>;
  }): any;

  /**
   * Evaluate a batch of flags
   *
   * @param requests Array of evaluation requests
   * @returns The batch evaluation response with all results (or Promise if async engine)
   * @throws Error if evaluation fails
   */
  evaluateBatch(
    requests: Array<{
      flagKey: string;
      entityId: string;
      context: Record<string, string>;
    }>
  ): any;

  /**
   * List all available flags
   *
   * @returns Array of available flags (or Promise if async engine)
   * @throws Error if listing fails
   */
  listFlags(): any;

  /**
   * Update the internal flag snapshot/state
   *
   * This method is used by the client to push new state data to the engine.
   * For WASM engines, this loads data into memory.
   * For REST engines, this is typically a no-op.
   *
   * @param data The snapshot data (flag definitions and state)
   */
  snapshot(data: any): void;

  /**
   * Check if this engine is asynchronous
   *
   * When true, all evaluation methods return Promises that must be awaited.
   * When false, all evaluation methods return results synchronously.
   *
   * @readonly
   */
  readonly isAsync?: boolean;
}

/**
 * Result wrapper for internal engine operations
 * Used by WASM engine to communicate status and errors
 *
 * @interface EngineResult
 */
export interface EngineResult<T> {
  status: 'success' | 'failure';
  result?: T;
  errorMessage: string;
}
