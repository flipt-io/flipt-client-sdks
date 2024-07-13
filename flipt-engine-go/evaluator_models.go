package flipt_engine_go

import (
	"time"

	"go.uber.org/zap/zapcore"
)

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

type Analytics struct {
	Timestamp       time.Time
	AnalyticName    string
	NamespaceKey    string
	FlagKey         string
	FlagType        flagType
	Reason          evalReason
	Match           *bool // server-side пишет пустую строку в тип BOOLEAN_FLAG_TYPE
	EvaluationValue string
	EntityID        string
	Value           uint32
}

func (r *Analytics) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddTime("timestamp", r.Timestamp)
	enc.AddString("analytic_name", r.AnalyticName)
	enc.AddString("namespace_key", r.NamespaceKey)
	enc.AddString("flag_key", r.FlagKey)
	enc.AddString("flag_type", r.FlagType.String())
	enc.AddString("reason", r.Reason.String())
	if r.Match != nil {
		enc.AddBool("match", *r.Match)
	}
	enc.AddString("evaluation_value", r.EvaluationValue)
	enc.AddString("entity_id", r.EntityID)
	enc.AddUint32("value", r.Value)

	return nil
}
