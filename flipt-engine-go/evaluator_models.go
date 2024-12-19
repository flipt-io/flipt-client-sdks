package flipt_engine_go

const (
	reasonDisabled evalReason = "FLAG_DISABLED_EVALUATION_REASON"
	reasonMatch    evalReason = "MATCH_EVALUATION_REASON"
	reasonDefault  evalReason = "DEFAULT_EVALUATION_REASON"
	reasonUnknown  evalReason = "UNKNOWN_EVALUATION_REASON"
	reasonError    evalReason = "ERROR_EVALUATION_REASON"

	errorReasonUnknown        errorEvalReason = "UNKNOWN_ERROR_EVALUATION_REASON"
	errorReasonNotFound       errorEvalReason = "NOT_FOUND_ERROR_EVALUATION_REASON"
	errorReasonNotImplemented errorEvalReason = "NOT_IMPLEMENTED_ERROR_EVALUATION_REASON"

	typeVariant evalType = "VARIANT_EVALUATION_RESPONSE_TYPE"
	typeBoolean evalType = "BOOLEAN_EVALUATION_RESPONSE_TYPE"
	typeError   evalType = "ERROR_EVALUATION_RESPONSE_TYPE"
)

type evalReason string

func (r evalReason) String() string {
	return string(r)
}

type errorEvalReason string

type evalType string

type ResponseFlag struct {
	Key          string   `json:"key"`
	Enabled      bool     `json:"enabled"`
	NamespaceKey string   `json:"namespaceKey"`
	Type         flagType `json:"type"`
}

type RequestEvaluation struct {
	NamespaceKey string            `json:"namespaceKey"`
	FlagKey      string            `json:"flagKey"`
	EntityID     string            `json:"entityId"`
	Context      map[string]string `json:"context"`
}

type ResponseEvaluation struct {
	Type            evalType         `json:"type"`
	ErrorResponse   *ResponseError   `json:"errorResponse,omitempty"`
	VariantResponse *ResponseVariant `json:"variantResponse,omitempty"`
	BooleanResponse *ResponseBoolean `json:"booleanResponse,omitempty"`
}

type ResponseError struct {
	FlagKey      string          `json:"flagKey"`
	NamespaceKey string          `json:"namespaceKey"`
	Reason       errorEvalReason `json:"reason"`
}

type ResponseVariant struct {
	FlagKey           string     `json:"flagKey"`
	Match             bool       `json:"match"`
	SegmentKeys       []string   `json:"segmentKeys"`
	Reason            evalReason `json:"reason"`
	VariantKey        string     `json:"variantKey"`
	VariantAttachment string     `json:"variantAttachment"`
}

type ResponseBoolean struct {
	FlagKey string     `json:"flagKey"`
	Enabled bool       `json:"enabled"`
	Reason  evalReason `json:"reason"`
}

type RequestBatchEvaluation struct {
	Requests []*RequestEvaluation `json:"requests"`
}

type ResponseBatchEvaluation struct {
	Responses []*ResponseEvaluation `json:"responses"`
}

type ResponseListFlags struct {
	Flags []*ResponseFlag `json:"flags"`
}
