// Re-export all types from core/types
export * from './core/types';

// Explicit exports for commonly used types (for better IDE autocomplete and documentation)
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
  ClientTokenAuthentication,
  JWTAuthentication,
  AuthenticationStrategy,
  IFetcher,
  IFetcherOptions,
  Hook
} from './core/types';

export { ErrorStrategy, ClientOptionsFactory } from './core/types';

// Re-export the base class
export { BaseFliptClient } from './core/base';

// Import implementations
import { FliptClient as NodeImpl } from './node';
import { FliptClient as BrowserImpl } from './browser';

// Export implementations with distinct names
export { NodeImpl as NodeFliptClient };
export { BrowserImpl as BrowserFliptClient };

// Create and export the default client based on environment
export const FliptClient =
  typeof window === 'undefined' ? NodeImpl : BrowserImpl;

// Export the type based on the runtime implementation
export type FliptClient = InstanceType<typeof FliptClient>;
