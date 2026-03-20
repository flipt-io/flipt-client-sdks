package venus

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	flipt "go.flipt.io/flipt-client"
)

func TestCasesWithClient(t *testing.T) {
	cases := loadCases(t)

	require.Greater(t, len(cases), 0, "No test cases found")

	opts := []flipt.Option{
		flipt.WithURL(FliptURL),
		flipt.WithRequestTimeout(30 * time.Second),
		flipt.WithUpdateInterval(10 * time.Minute),
		flipt.WithFetchMode(flipt.FetchModePolling),
		flipt.WithEnvironment("default"),
		flipt.WithNamespace("venus"),
	}

	if token := os.Getenv("FLIPT_AUTH_TOKEN"); token != "" {
		opts = append(opts, flipt.WithClientTokenAuthentication(token))
	}

	client, err := flipt.NewClient(t.Context(), opts...)
	require.NoError(t, err)
	defer func() { _ = client.Close(t.Context()) }()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			switch tc.Type {
			case "variant":
				testVariantWithClient(t, client, tc)
			case "boolean":
				testBooleanWithClient(t, client, tc)
			case "batch":
				t.Skip("Batch evaluation not supported by client SDK yet")
			default:
				require.Fail(t, "Unknown test type: %s", tc.Type)
			}
		})
	}
}

func testVariantWithClient(t *testing.T, client *flipt.Client, tc TestCase) {
	t.Helper()
	result, err := client.EvaluateVariant(t.Context(), &flipt.EvaluationRequest{
		FlagKey:  tc.Request.FlagKey,
		EntityID: tc.Request.EntityID,
		Context:  tc.Request.Context,
	})
	require.NoError(t, err)

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

func testBooleanWithClient(t *testing.T, client *flipt.Client, tc TestCase) {
	t.Helper()
	result, err := client.EvaluateBoolean(t.Context(), &flipt.EvaluationRequest{
		FlagKey:  tc.Request.FlagKey,
		EntityID: tc.Request.EntityID,
		Context:  tc.Request.Context,
	})
	require.NoError(t, err)

	expected := tc.Expected

	require.Equal(t, expected.FlagKey, result.FlagKey)
	require.Equal(t, expected.Reason, result.Reason)
	require.Equal(t, expected.SegmentKeys, result.SegmentKeys)
	if expected.Enabled != nil {
		require.Equal(t, *expected.Enabled, result.Enabled)
	}
}
