package venus

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.flipt.io/flipt/rpc/flipt/evaluation"
	sdk "go.flipt.io/flipt/sdk/go"
	"go.flipt.io/flipt/sdk/go/http"
)

func TestCasesWithSDK(t *testing.T) {
	cases := loadCases(t)

	require.Greater(t, len(cases), 0, "No test cases found")

	opts := []sdk.Option{}
	if token := os.Getenv("FLIPT_AUTH_TOKEN"); token != "" {
		opts = append(opts, sdk.WithAuthenticationProvider(sdk.StaticTokenAuthenticationProvider(token)))
	}

	transport := http.NewTransport(FliptURL)
	client := sdk.New(transport, opts...)

	eval := client.Evaluation()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			switch tc.Type {
			case "variant":
				testVariantWithSDK(t, eval, tc)
			case "boolean":
				testBooleanWithSDK(t, eval, tc)
			case "batch":
				testBatchWithSDK(t, eval, tc)
			default:
				require.Fail(t, "Unknown test type: %s", tc.Type)
			}
		})
	}
}

func testVariantWithSDK(t *testing.T, eval *sdk.Evaluation, tc TestCase) {
	t.Helper()
	result, err := eval.Variant(t.Context(), &evaluation.EvaluationRequest{
		NamespaceKey: tc.Namespace,
		FlagKey:      tc.Request.FlagKey,
		EntityId:     tc.Request.EntityID,
		Context:      tc.Request.Context,
	})
	require.NoError(t, err)

	expected := tc.Expected

	require.Equal(t, expected.FlagKey, result.FlagKey)
	require.Equal(t, expected.Reason, result.Reason.String())

	segmentKeys := result.GetSegmentKeys()
	if segmentKeys == nil {
		segmentKeys = []string{}
	}
	require.Equal(t, expected.SegmentKeys, segmentKeys)
	if expected.Match != nil {
		require.Equal(t, *expected.Match, result.Match)
	}
	if expected.VariantKey != nil {
		require.Equal(t, *expected.VariantKey, result.VariantKey)
	}
}

func testBooleanWithSDK(t *testing.T, eval *sdk.Evaluation, tc TestCase) {
	t.Helper()
	result, err := eval.Boolean(t.Context(), &evaluation.EvaluationRequest{
		NamespaceKey: tc.Namespace,
		FlagKey:      tc.Request.FlagKey,
		EntityId:     tc.Request.EntityID,
		Context:      tc.Request.Context,
	})
	require.NoError(t, err)

	expected := tc.Expected

	require.Equal(t, expected.FlagKey, result.FlagKey)
	require.Equal(t, expected.Reason, result.Reason.String())
	if expected.Enabled != nil {
		require.Equal(t, *expected.Enabled, result.Enabled)
	}
}

func testBatchWithSDK(t *testing.T, eval *sdk.Evaluation, tc TestCase) {
	requests := make([]*evaluation.EvaluationRequest, len(tc.Requests))
	for i, r := range tc.Requests {
		requests[i] = &evaluation.EvaluationRequest{
			NamespaceKey: tc.Namespace,
			FlagKey:      r.FlagKey,
			EntityId:     r.EntityID,
			Context:      r.Context,
		}
	}

	result, err := eval.Batch(t.Context(), &evaluation.BatchEvaluationRequest{
		Requests: requests,
	})
	require.NoError(t, err)

	expected := tc.Expected

	require.Equal(t, len(expected.Responses), len(result.Responses))

	for i, resp := range result.Responses {
		exp := expected.Responses[i]

		require.Equal(t, exp.Type, resp.Type.String())

		switch {
		case resp.Type == evaluation.EvaluationResponseType_VARIANT_EVALUATION_RESPONSE_TYPE && exp.Type == "VARIANT_EVALUATION_RESPONSE_TYPE":
			variantResp := resp.GetVariantResponse()
			require.NotNil(t, variantResp)
			expVariant := exp.VariantEvaluationResponse
			require.NotNil(t, expVariant)
			require.Equal(t, expVariant.FlagKey, variantResp.FlagKey)
			require.Equal(t, expVariant.Reason, variantResp.Reason.String())
			segKeys := variantResp.SegmentKeys
			if segKeys == nil {
				segKeys = []string{}
			}
			require.Equal(t, expVariant.SegmentKeys, segKeys)
			require.Equal(t, expVariant.Match, variantResp.Match)
			require.Equal(t, expVariant.VariantKey, variantResp.VariantKey)

		case resp.Type == evaluation.EvaluationResponseType_BOOLEAN_EVALUATION_RESPONSE_TYPE && exp.Type == "BOOLEAN_EVALUATION_RESPONSE_TYPE":
			boolResp := resp.GetBooleanResponse()
			require.NotNil(t, boolResp)
			expBool := exp.BooleanEvaluationResponse
			require.NotNil(t, expBool)
			require.Equal(t, expBool.FlagKey, boolResp.FlagKey)
			require.Equal(t, expBool.Reason, boolResp.Reason.String())
			require.Equal(t, expBool.Enabled, boolResp.Enabled)

		case resp.Type == evaluation.EvaluationResponseType_ERROR_EVALUATION_RESPONSE_TYPE && exp.Type == "ERROR_EVALUATION_RESPONSE_TYPE":
			errResp := resp.GetErrorResponse()
			require.NotNil(t, errResp)
			expErr := exp.ErrorEvaluationResponse
			require.NotNil(t, expErr)
			require.Equal(t, expErr.FlagKey, errResp.FlagKey)
			require.Equal(t, expErr.Reason, errResp.Reason.String())
		}
	}
}
