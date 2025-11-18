/**
 * Re-export all types for users who prefer explicit type imports.
 *
 * @example
 * ```typescript
 * import type { Flag, ClientOptions } from '@flipt-io/flipt-client-js/types';
 * ```
 */

export type {
  Flag,
  ClientOptions,
  EvaluationRequest,
  VariantEvaluationResponse,
  BooleanEvaluationResponse,
  BatchEvaluationResponse,
  ErrorEvaluationResponse,
  EvaluationResponse,
  VariantResult,
  BooleanResult,
  BatchResult,
  ListFlagsResult,
  Result,
  Response,
  ClientTokenAuthentication,
  JWTAuthentication,
  AuthenticationStrategy,
  IFetcher,
  IFetcherOptions,
  Hook
} from './core/types';

export { ErrorStrategy, ClientOptionsFactory } from './core/types';
