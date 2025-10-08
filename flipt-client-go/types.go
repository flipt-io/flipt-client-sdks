package flipt

// EvaluationRequest represents the request structure for evaluating a flag.
type EvaluationRequest struct {
	FlagKey  string            `json:"flag_key"`
	EntityID string            `json:"entity_id"`
	Context  map[string]string `json:"context"`
}

// clientTokenAuthentication is used for client token authentication.
type clientTokenAuthentication struct {
	Token string `json:"client_token"`
}

// jwtAuthentication is used for JWT authentication.
type jwtAuthentication struct {
	Token string `json:"jwt_token"`
}

// FetchMode determines how the client fetches flag state.
type FetchMode string

const (
	// FetchModeStreaming uses streaming to fetch flag state.
	FetchModeStreaming FetchMode = "streaming"
	// FetchModePolling uses polling to fetch flag state.
	FetchModePolling FetchMode = "polling"
)

// ErrorStrategy determines how the client handles errors when fetching flag state.
type ErrorStrategy string

const (
	// ErrorStrategyFail causes the client to return an error if flag state cannot be fetched.
	ErrorStrategyFail ErrorStrategy = "fail"
	// ErrorStrategyFallback causes the client to use the last known good state if an error occurs.
	ErrorStrategyFallback ErrorStrategy = "fallback"
)

// Flag represents a feature flag.
type Flag struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
}

// VariantEvaluationResponse is the response for a variant flag evaluation.
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

// BooleanEvaluationResponse is the response for a boolean flag evaluation.
type BooleanEvaluationResponse struct {
	Enabled               bool     `json:"enabled"`
	FlagKey               string   `json:"flag_key"`
	Reason                string   `json:"reason"`
	RequestDurationMillis float64  `json:"request_duration_millis"`
	Timestamp             string   `json:"timestamp"`
	SegmentKeys           []string `json:"segment_keys"`
}

// ErrorEvaluationResponse is the response for an error during flag evaluation.
type ErrorEvaluationResponse struct {
	FlagKey      string `json:"flag_key"`
	NamespaceKey string `json:"namespace_key"`
	Reason       string `json:"reason"`
}

// BatchEvaluationResponse is the response for a batch flag evaluation.
type BatchEvaluationResponse struct {
	Responses             []*Response `json:"responses"`
	RequestDurationMillis float64     `json:"request_duration_millis"`
}

// Response is a wrapper for different types of evaluation responses.
type Response struct {
	Type                      string                     `json:"type"`
	VariantEvaluationResponse *VariantEvaluationResponse `json:"variant_evaluation_response,omitempty"`
	BooleanEvaluationResponse *BooleanEvaluationResponse `json:"boolean_evaluation_response,omitempty"`
	ErrorEvaluationResponse   *ErrorEvaluationResponse   `json:"error_evaluation_response,omitempty"`
}

// Result is a generic result wrapper for evaluation responses.
type Result[R any] struct {
	Status       string `json:"status"`
	Result       *R     `json:"result,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// VariantResult is a result wrapper for VariantEvaluationResponse.
type VariantResult Result[VariantEvaluationResponse]

// BooleanResult is a result wrapper for BooleanEvaluationResponse.
type BooleanResult Result[BooleanEvaluationResponse]

// BatchResult is a result wrapper for BatchEvaluationResponse.
type BatchResult Result[BatchEvaluationResponse]

// ListFlagsResult is a result wrapper for a list of Flag.
type ListFlagsResult Result[[]Flag]
