package venus

import (
	"cmp"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	FliptURL  = cmp.Or(os.Getenv("FLIPT_URL"), "http://localhost:8080")
	Namespace = "venus"
	CasesFile = "cases.json"
)

type TestCase struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Type      string    `json:"type"`
	Request   Request   `json:"request"`
	Requests  []Request `json:"requests,omitempty"`
	Expected  Expected  `json:"expected"`
}

type Request struct {
	FlagKey  string            `json:"flag_key"`
	EntityID string            `json:"entity_id"`
	Context  map[string]string `json:"context"`
}

type Expected struct {
	Match             *bool              `json:"match,omitempty"`
	Enabled           *bool              `json:"enabled,omitempty"`
	SegmentKeys       []string           `json:"segment_keys"`
	Reason            string             `json:"reason"`
	FlagKey           string             `json:"flag_key"`
	VariantKey        *string            `json:"variant_key"`
	VariantAttachment any                `json:"variant_attachment"`
	Responses         []BatchResponseAPI `json:"responses,omitempty"`
}

type BatchResponseAPI struct {
	Type                      string                  `json:"type"`
	BooleanEvaluationResponse *BoolEvalResponseAPI    `json:"booleanResponse"`
	VariantEvaluationResponse *VariantEvalResponseAPI `json:"variantResponse"`
	ErrorEvaluationResponse   *ErrorEvalResponseAPI   `json:"errorResponse"`
}

type BoolEvalResponseAPI struct {
	Enabled     bool     `json:"enabled"`
	FlagKey     string   `json:"flagKey"`
	Reason      string   `json:"reason"`
	SegmentKeys []string `json:"segmentKeys"`
}

type VariantEvalResponseAPI struct {
	Match             bool     `json:"match"`
	SegmentKeys       []string `json:"segmentKeys"`
	Reason            string   `json:"reason"`
	FlagKey           string   `json:"flagKey"`
	VariantKey        string   `json:"variantKey"`
	VariantAttachment *string  `json:"variantAttachment"`
}

type ErrorEvalResponseAPI struct {
	FlagKey      string `json:"flagKey"`
	NamespaceKey string `json:"namespaceKey"`
	Reason       string `json:"reason"`
}

func loadCases(t *testing.T) []TestCase {
	t.Helper()

	data, err := os.ReadFile(CasesFile)
	require.NoError(t, err, "Failed to read cases.json")

	var cases []TestCase
	err = json.Unmarshal(data, &cases)
	require.NoError(t, err, "Failed to parse cases.json")

	return cases
}
