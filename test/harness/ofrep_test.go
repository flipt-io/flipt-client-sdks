package venus

import (
	"os"
	"testing"

	"github.com/open-feature/go-sdk-contrib/providers/ofrep"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/stretchr/testify/require"
)

func TestCasesWithOFREP(t *testing.T) {
	cases := loadCases(t)

	require.Greater(t, len(cases), 0, "No test cases found")

	opts := []ofrep.Option{
		ofrep.WithHeaderProvider(func() (string, string) { return "X-Flipt-Namespace", "venus" }),
	}
	if token := os.Getenv("FLIPT_AUTH_TOKEN"); token != "" {
		opts = append(opts, ofrep.WithBearerToken(token))
	}

	provider := ofrep.NewProvider(FliptURL+"/", opts...)
	err := openfeature.SetProviderAndWait(provider)
	require.NoError(t, err)
	defer openfeature.Shutdown()

	client := openfeature.NewClient("venus-test")

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			switch tc.Type {
			case "variant":
				testVariantWithOFREP(t, client, tc)
			case "boolean":
				testBooleanWithOFREP(t, client, tc)
			case "batch":
				t.Skip("OFREP batch evaluation uses bulk provider with AllFlagsValue which is not yet available")
			default:
				require.Fail(t, "Unknown test type: %s", tc.Type)
			}
		})
	}
}

func testVariantWithOFREP(t *testing.T, client *openfeature.Client, tc TestCase) {
	t.Helper()

	ctxMap := make(map[string]any)
	for k, v := range tc.Request.Context {
		ctxMap[k] = v
	}
	evalCtx := openfeature.NewEvaluationContext(tc.Request.EntityID, ctxMap)

	result, err := client.StringValue(t.Context(), tc.Request.FlagKey, "", evalCtx)
	require.NoError(t, err)

	expected := tc.Expected

	if expected.VariantKey != nil {
		require.Equal(t, *expected.VariantKey, result)
	}
}

func testBooleanWithOFREP(t *testing.T, client *openfeature.Client, tc TestCase) {
	t.Helper()

	ctxMap := make(map[string]any)
	for k, v := range tc.Request.Context {
		ctxMap[k] = v
	}
	evalCtx := openfeature.NewEvaluationContext(tc.Request.EntityID, ctxMap)

	result, err := client.BooleanValue(t.Context(), tc.Request.FlagKey, false, evalCtx)
	require.NoError(t, err)

	expected := tc.Expected

	if expected.Enabled != nil {
		require.Equal(t, *expected.Enabled, result)
	}
}
