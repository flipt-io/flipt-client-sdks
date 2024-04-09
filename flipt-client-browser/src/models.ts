export interface AuthenticationStrategy {}

export interface ClientTokenAuthentication extends AuthenticationStrategy {
  client_token: string;
}

export interface JWTAuthentication extends AuthenticationStrategy {
  jwt_token: string;
}

export interface EngineOpts {
  url?: string;
  authentication?: AuthenticationStrategy;
  reference?: string;
  fetcher?: () => Promise<Response>;
}

export interface EvaluationRequest {
  flag_key: string;
  entity_id: string;
  context: object;
}

export interface VariantEvaluationResponse {
  match: boolean;
  segment_keys: string[];
  reason: string;
  flag_key: string;
  variant_key: string;
  variant_attachment: string;
  request_duration_millis: number;
  timestamp: string;
}

export interface BooleanEvaluationResponse {
  enabled: boolean;
  flag_key: string;
  reason: string;
  request_duration_millis: number;
  timestamp: string;
}

export interface ErrorEvaluationResponse {
  flag_key: string;
  namespace_key: string;
  reason: string;
}

export interface EvaluationResponse {
  type: string;
  boolean_evaluation_response?: BooleanEvaluationResponse;
  variant_evaluation_response?: VariantEvaluationResponse;
  error_evaluation_response?: ErrorEvaluationResponse;
}

export interface BatchEvaluationResponse {
  responses: EvaluationResponse[];
  request_duration_millis: number;
}

export interface Result<T> {
  status: string;
  result?: T;
  error_message: string;
}

export interface VariantResult extends Result<VariantEvaluationResponse> {}

export interface BooleanResult extends Result<BooleanEvaluationResponse> {}

export interface BatchResult extends Result<BatchEvaluationResponse> {}
