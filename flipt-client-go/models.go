package evaluation

type evaluationRequest struct {
	NamespaceKey string            `json:"namespace_key"`
	FlagKey      string            `json:"flag_key"`
	EntityId     string            `json:"entity_id"`
	Context      map[string]string `json:"context"`
}

type clientTokenAuthentication struct {
	Token string `json:"client_token"`
}

type jwtAuthentication struct {
	Token string `json:"jwt_token"`
}

type EngineOpts[T any] struct {
	URL            string `json:"url,omitempty"`
	Authentication *T     `json:"authentication,omitempty"`
	UpdateInterval int    `json:"update_interval,omitempty"`
	Reference      string `json:"reference,omitempty"`
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

type Result[R any] struct {
	Status       string `json:"status"`
	Result       *R     `json:"result,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type VariantResult Result[VariantEvaluationResponse]

type BooleanResult Result[BooleanEvaluationResponse]
