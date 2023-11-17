interface EvaluationRequest {
  namespace_key: string;
  flag_key: string;
  entity_id: string;
  context: string;
}

interface VariantEvaluationResponse {
  match: boolean;
  segment_keys: string[];
  reason: string;
  flag_key: string;
  variant_key: string;
  variant_attachment: string;
  request_duration_millis: number;
  timestamp: string;
}

interface BooleanEvaluationResponse {
  enabled: boolean;
  flag_key: string;
  reason: string;
  request_duration_millis: number;
  timestamp: string;
}

interface VariantResult {
  status: string;
  result: VariantEvaluationResponse;
  error_message: string;
}

interface BooleanResult {
  status: string;
  result: BooleanEvaluationResponse;
  error_message: string;
}

export { BooleanResult, EvaluationRequest, VariantResult };
