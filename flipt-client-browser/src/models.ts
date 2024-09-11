export interface IFetcherOptions {
  etag?: string;
}

export interface IFetcher {
  (opts?: IFetcherOptions): Promise<Response>;
}

export interface AuthenticationStrategy {}

export interface ClientTokenAuthentication extends AuthenticationStrategy {
  clientToken: string;
}

export interface JWTAuthentication extends AuthenticationStrategy {
  jwtToken: string;
}

export interface ClientOptions {
  url?: string;
  authentication?: AuthenticationStrategy;
  reference?: string;
  fetcher?: IFetcher;
}

export interface EvaluationRequest {
  flagKey: string;
  entityId: string;
  context: object;
}

export interface VariantEvaluationResponse {
  match: boolean;
  segmentKeys: string[];
  reason: string;
  flagKey: string;
  variantKey?: string;
  variantAttachment?: string;
  requestDurationMillis: number;
  timestamp: string;
}

export interface BooleanEvaluationResponse {
  enabled: boolean;
  flagKey: string;
  reason: string;
  requestDurationMillis: number;
  timestamp: string;
}

export interface ErrorEvaluationResponse {
  flagKey: string;
  namespaceKey: string;
  reason: string;
}

export interface EvaluationResponse {
  type: string;
  booleanEvaluationResponse?: BooleanEvaluationResponse;
  variantEvaluationResponse?: VariantEvaluationResponse;
  errorEvaluationResponse?: ErrorEvaluationResponse;
}

export interface BatchEvaluationResponse {
  responses: EvaluationResponse[];
  requestDurationMillis: number;
}

export interface Result<T> {
  status: 'success' | 'failure';
  result?: T;
  errorMessage: string;
}

export interface VariantResult extends Result<VariantEvaluationResponse> {}

export interface BooleanResult extends Result<BooleanEvaluationResponse> {}

export interface BatchResult extends Result<BatchEvaluationResponse> {}
