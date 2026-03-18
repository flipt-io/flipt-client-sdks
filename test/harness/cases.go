package venus

import (
	"cmp"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var (
	FliptURL  = cmp.Or(os.Getenv("FLIPT_URL"), "http://localhost:8080")
	Namespace = "venus"
	CasesFile = "cases.yaml"
)

type TestCase struct {
	Name      string    `yaml:"name"`
	Namespace string    `yaml:"namespace"`
	Type      string    `yaml:"type"`
	Request   Request   `yaml:"request"`
	Requests  []Request `yaml:"requests,omitempty"`
	Expected  Expected  `yaml:"expected"`
}

type Request struct {
	FlagKey  string            `yaml:"flag_key"`
	EntityID string            `yaml:"entity_id"`
	Context  map[string]string `yaml:"context"`
}

type Expected struct {
	Match             *bool              `yaml:"match,omitempty"`
	Enabled           *bool              `yaml:"enabled,omitempty"`
	SegmentKeys       []string           `yaml:"segment_keys"`
	Reason            string             `yaml:"reason"`
	FlagKey           string             `yaml:"flag_key"`
	VariantKey        *string            `yaml:"variant_key"`
	VariantAttachment any                `yaml:"variant_attachment"`
	Responses         []BatchResponseAPI `yaml:"responses,omitempty"`
}

type BatchResponseAPI struct {
	Type                      string                  `yaml:"type"`
	BooleanEvaluationResponse *BoolEvalResponseAPI    `yaml:"booleanResponse"`
	VariantEvaluationResponse *VariantEvalResponseAPI `yaml:"variantResponse"`
	ErrorEvaluationResponse   *ErrorEvalResponseAPI   `yaml:"errorResponse"`
}

type BoolEvalResponseAPI struct {
	Enabled     bool     `yaml:"enabled"`
	FlagKey     string   `yaml:"flagKey"`
	Reason      string   `yaml:"reason"`
	SegmentKeys []string `yaml:"segmentKeys"`
}

type VariantEvalResponseAPI struct {
	Match             bool     `yaml:"match"`
	SegmentKeys       []string `yaml:"segmentKeys"`
	Reason            string   `yaml:"reason"`
	FlagKey           string   `yaml:"flagKey"`
	VariantKey        string   `yaml:"variantKey"`
	VariantAttachment *string  `yaml:"variantAttachment"`
}

type ErrorEvalResponseAPI struct {
	FlagKey      string `yaml:"flagKey"`
	NamespaceKey string `yaml:"namespaceKey"`
	Reason       string `yaml:"reason"`
}

func loadCases(t *testing.T) []TestCase {
	t.Helper()

	data, err := os.ReadFile(CasesFile)
	require.NoError(t, err, "Failed to read cases.yaml")

	var cases []TestCase
	err = yaml.Unmarshal(data, &cases)
	require.NoError(t, err, "Failed to parse cases.yaml")

	return cases
}
