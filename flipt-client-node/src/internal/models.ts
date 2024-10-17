import {
  Flag,
  VariantEvaluationResponse,
  BooleanEvaluationResponse,
  BatchEvaluationResponse
} from '../models';
/**
 * Represents a specialized result object for variant evaluation, extending
 * the generic Result interface with a specific type VariantEvaluationResponse.
 */
export interface VariantResult extends Result<VariantEvaluationResponse> {}

/**
 * Represents a specialized result object for boolean evaluation, extending
 * the generic Result interface with a specific type BooleanEvaluationResponse.
 */
export interface BooleanResult extends Result<BooleanEvaluationResponse> {}

/**
 * Represents a specialized result object for batch evaluation, extending
 * the generic Result interface with a specific type BatchEvaluationResponse.
 */
export interface BatchResult extends Result<BatchEvaluationResponse> {}

/**
 * Represents a specialized result object for listing flags, extending
 * the generic Result interface with a specific type ListFlagsResponse.
 */
export interface ListFlagsResult extends Result<Flag[]> {}

export interface Result<T> {
  /** Status of the result - `success` or `failure`. */
  status: string;
  /** Actual result of type T if the operation was successful. */
  result?: T;
  /** Error message describing the reason for failure, if applicable.*/
  errorMessage: string;
}

export interface StreamChunk {
  result: StreamResult;
}

export interface StreamResult {
  namespaces: Record<string, any>;
}
