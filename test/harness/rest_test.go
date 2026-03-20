package venus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type VariantResponse struct {
	Match             bool     `json:"match"`
	SegmentKeys       []string `json:"segmentKeys"`
	Reason            string   `json:"reason"`
	FlagKey           string   `json:"flagKey"`
	VariantKey        string   `json:"variantKey"`
	VariantAttachment *string  `json:"variantAttachment"`
}

type BooleanResponse struct {
	Enabled     bool     `json:"enabled"`
	FlagKey     string   `json:"flagKey"`
	Reason      string   `json:"reason"`
	SegmentKeys []string `json:"segmentKeys"`
}

type BatchResponseWrapper struct {
	Responses []BatchResponseAPI `json:"responses"`
}

func TestCases(t *testing.T) {
	cases := loadCases(t)

	require.Greater(t, len(cases), 0, "No test cases found")

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			switch tc.Type {
			case "variant":
				testVariant(t, tc)
			case "boolean":
				testBoolean(t, tc)
			case "batch":
				testBatch(t, tc)
			default:
				require.Fail(t, "Unknown test type: %s", tc.Type)
			}
		})
	}
}

func testVariant(t *testing.T, tc TestCase) {
	t.Helper()

	result := doEvaluateRequest(t, tc.Namespace, tc.Request.FlagKey, tc.Request.EntityID, tc.Request.Context, "variant")

	expected := tc.Expected

	require.Equal(t, expected.FlagKey, result.FlagKey)
	require.Equal(t, expected.Reason, result.Reason)
	require.Equal(t, expected.SegmentKeys, result.SegmentKeys)
	if expected.Match != nil {
		require.Equal(t, *expected.Match, result.Match)
	}
	if expected.VariantKey != nil {
		require.Equal(t, *expected.VariantKey, result.VariantKey)
	}
}

func testBoolean(t *testing.T, tc TestCase) {
	t.Helper()

	result := doBooleanRequest(t, tc.Namespace, tc.Request.FlagKey, tc.Request.EntityID, tc.Request.Context)

	expected := tc.Expected

	require.Equal(t, expected.FlagKey, result.FlagKey)
	require.Equal(t, expected.Reason, result.Reason)
	require.Equal(t, expected.SegmentKeys, result.SegmentKeys)
	if expected.Enabled != nil {
		require.Equal(t, *expected.Enabled, result.Enabled)
	}
}

func testBatch(t *testing.T, tc TestCase) {
	t.Helper()

	result := doBatchRequest(t, tc)

	expected := tc.Expected

	require.Equal(t, len(expected.Responses), len(result.Responses))

	for i, resp := range result.Responses {
		exp := expected.Responses[i]

		require.Equal(t, exp.Type, resp.Type)

		switch {
		case resp.Type == "VARIANT_EVALUATION_RESPONSE_TYPE" && exp.Type == "VARIANT_EVALUATION_RESPONSE_TYPE":
			require.NotNil(t, resp.VariantEvaluationResponse)
			require.NotNil(t, exp.VariantEvaluationResponse)
			compareVariantResponse(t, i, resp.VariantEvaluationResponse, exp.VariantEvaluationResponse)

		case resp.Type == "BOOLEAN_EVALUATION_RESPONSE_TYPE" && exp.Type == "BOOLEAN_EVALUATION_RESPONSE_TYPE":
			require.NotNil(t, resp.BooleanEvaluationResponse)
			require.NotNil(t, exp.BooleanEvaluationResponse)
			compareBooleanResponse(t, i, resp.BooleanEvaluationResponse, exp.BooleanEvaluationResponse)

		case resp.Type == "ERROR_EVALUATION_RESPONSE_TYPE" && exp.Type == "ERROR_EVALUATION_RESPONSE_TYPE":
			require.NotNil(t, resp.ErrorEvaluationResponse)
			require.NotNil(t, exp.ErrorEvaluationResponse)
			compareErrorResponse(t, i, resp.ErrorEvaluationResponse, exp.ErrorEvaluationResponse)
		}
	}
}

func doEvaluateRequest(t *testing.T, namespace, flagKey, entityID string, context map[string]string, endpoint string) VariantResponse {
	t.Helper()

	url := fmt.Sprintf("%s/evaluate/v1/%s", FliptURL, endpoint)

	reqBody := map[string]any{
		"namespaceKey":   namespace,
		"environmentKey": "default",
		"flagKey":        flagKey,
		"entityId":       entityID,
		"context":        context,
	}

	resp := doRequest(t, url, reqBody)

	var result VariantResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
}

func doBooleanRequest(t *testing.T, namespace, flagKey, entityID string, context map[string]string) BooleanResponse {
	t.Helper()

	url := fmt.Sprintf("%s/evaluate/v1/boolean", FliptURL)

	reqBody := map[string]any{
		"namespaceKey":   namespace,
		"environmentKey": "default",
		"flagKey":        flagKey,
		"entityId":       entityID,
		"context":        context,
	}

	resp := doRequest(t, url, reqBody)

	var result BooleanResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
}

func doBatchRequest(t *testing.T, tc TestCase) BatchResponseWrapper {
	t.Helper()

	url := fmt.Sprintf("%s/evaluate/v1/batch", FliptURL)

	requests := make([]map[string]any, len(tc.Requests))
	for i, r := range tc.Requests {
		requests[i] = map[string]any{
			"namespaceKey":   tc.Namespace,
			"environmentKey": "default",
			"flagKey":        r.FlagKey,
			"entityId":       r.EntityID,
			"context":        r.Context,
		}
	}

	resp := doRequest(t, url, map[string]any{"requests": requests})

	var result BatchResponseWrapper
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
}

func doRequest(t *testing.T, url string, reqBody map[string]any) *http.Response {
	t.Helper()

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), "POST", url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token := os.Getenv("FLIPT_AUTH_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	return resp
}

func compareVariantResponse(t *testing.T, i int, got, exp *VariantEvalResponseAPI) {
	require.Equal(t, exp.FlagKey, got.FlagKey, "Response[%d] FlagKey", i)
	require.Equal(t, exp.Reason, got.Reason, "Response[%d] Reason", i)
	require.Equal(t, exp.SegmentKeys, got.SegmentKeys, "Response[%d] SegmentKeys", i)
	require.Equal(t, exp.Match, got.Match, "Response[%d] Match", i)
	require.Equal(t, exp.VariantKey, got.VariantKey, "Response[%d] VariantKey", i)
}

func compareBooleanResponse(t *testing.T, i int, got, exp *BoolEvalResponseAPI) {
	require.Equal(t, exp.FlagKey, got.FlagKey, "Response[%d] FlagKey", i)
	require.Equal(t, exp.Reason, got.Reason, "Response[%d] Reason", i)
	require.Equal(t, exp.SegmentKeys, got.SegmentKeys, "Response[%d] SegmentKeys", i)
	require.Equal(t, exp.Enabled, got.Enabled, "Response[%d] Enabled", i)
}

func compareErrorResponse(t *testing.T, i int, got, exp *ErrorEvalResponseAPI) {
	require.Equal(t, exp.FlagKey, got.FlagKey, "Response[%d] FlagKey", i)
	require.Equal(t, exp.NamespaceKey, got.NamespaceKey, "Response[%d] NamespaceKey", i)
	require.True(t, got.Reason == exp.Reason || strings.Contains(got.Reason, exp.Reason), "Response[%d] Reason", i)
}
