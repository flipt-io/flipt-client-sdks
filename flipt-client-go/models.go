package evaluation

type EvaluationRequest struct {
	FlagKey  string            `json:"flag_key"`
	EntityId string            `json:"entity_id"`
	Context  map[string]string `json:"context"`
}

type clientTokenAuthentication struct {
	Token string `json:"client_token"`
}

type jwtAuthentication struct {
	Token string `json:"jwt_token"`
}

type FetchMode string

const (
	FetchModeStreaming FetchMode = "streaming"
	FetchModePolling   FetchMode = "polling"
)

type ErrorStrategy string

const (
	ErrorStrategyFail     ErrorStrategy = "fail"
	ErrorStrategyFallback ErrorStrategy = "fallback"
)

type Flag struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
}

type VariantEvaluationResponse struct {
	Match                 bool     `json:"match"`
	SegmentKeys           []string `json:"segment_keys"`
	Reason                string   `json:"reason"`
	FlagKey               string   `json:"flag_key"`
	VariantKey            string   `json:"variant_key"`
	VariantAttachment     string   `json:"variant_attachment"`
	RequestDurationMillis float64  `json:"request_duration_millis"`
	Timestamp             string   `json:"timestamp"`
}

type BooleanEvaluationResponse struct {
	Enabled               bool    `json:"enabled"`
	FlagKey               string  `json:"flag_key"`
	Reason                string  `json:"reason"`
	RequestDurationMillis float64 `json:"request_duration_millis"`
	Timestamp             string  `json:"timestamp"`
}

type ErrorEvaluationResponse struct {
	FlagKey      string `json:"flag_key"`
	NamespaceKey string `json:"namespace_key"`
	Reason       string `json:"reason"`
}

type BatchEvaluationResponse struct {
	Responses             []*Response `json:"responses"`
	RequestDurationMillis float64     `json:"request_duration_millis"`
}

type Response struct {
	Type                      string                     `json:"type"`
	VariantEvaluationResponse *VariantEvaluationResponse `json:"variant_evaluation_response,omitempty"`
	BooleanEvaluationResponse *BooleanEvaluationResponse `json:"boolean_evaluation_response,omitempty"`
	ErrorEvaluationResponse   *ErrorEvaluationResponse   `json:"error_evaluation_response,omitempty"`
}

type Result[R any] struct {
	Status       string `json:"status"`
	Result       *R     `json:"result,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type VariantResult Result[VariantEvaluationResponse]

type BooleanResult Result[BooleanEvaluationResponse]

type BatchResult Result[BatchEvaluationResponse]

type ListFlagsResult Result[[]Flag]
