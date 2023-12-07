package evaluation_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	evaluation "go.flipt.io/flipt/flipt-client-go"
)

var fliptUrl string
var authToken string

func init() {
	fliptUrl = os.Getenv("FLIPT_URL")
	if fliptUrl == "" {
		panic("set FLIPT_URL")
	}

	authToken = os.Getenv("FLIPT_AUTH_TOKEN")
	if authToken == "" {
		panic("set FLIPT_AUTH_TOKEN")
	}
}

func TestVariant(t *testing.T) {
	evaluationClient, err := evaluation.NewClient(evaluation.WithURL(fliptUrl), evaluation.WithAuthToken(authToken))
	require.NoError(t, err)

	variant, err := evaluationClient.EvaluateVariant(context.TODO(), "flag1", "someentity", map[string]string{
		"fizz": "buzz",
	})
	require.NoError(t, err)

	assert.Equal(t, "success", variant.Status)
	assert.Empty(t, variant.ErrorMessage)
	assert.True(t, variant.Result.Match)
	assert.Equal(t, "flag1", variant.Result.FlagKey)
	assert.Equal(t, "MATCH_EVALUATION_REASON", variant.Result.Reason)
	assert.Contains(t, variant.Result.SegmentKeys, "segment1")
}

func TestBoolean(t *testing.T) {
	evaluationClient, err := evaluation.NewClient(evaluation.WithURL(fliptUrl), evaluation.WithAuthToken(authToken))
	require.NoError(t, err)

	boolean, err := evaluationClient.EvaluateBoolean(context.TODO(), "flag_boolean", "someentity", map[string]string{
		"fizz": "buzz",
	})
	require.NoError(t, err)

	assert.Equal(t, "success", boolean.Status)
	assert.Empty(t, boolean.ErrorMessage)
	assert.Equal(t, "flag_boolean", boolean.Result.FlagKey)
	assert.True(t, boolean.Result.Enabled)
	assert.Equal(t, "MATCH_EVALUATION_REASON", boolean.Result.Reason)
}

func TestVariantFailure(t *testing.T) {
	evaluationClient, err := evaluation.NewClient(evaluation.WithURL(fliptUrl), evaluation.WithAuthToken(authToken))
	require.NoError(t, err)

	variant, err := evaluationClient.EvaluateVariant(context.TODO(), "nonexistent", "someentity", map[string]string{
		"fizz": "buzz",
	})
	require.NoError(t, err)

	assert.Nil(t, variant.Result)
	assert.Equal(t, "failure", variant.Status)
	assert.Equal(t, "failed to get flag information default/nonexistent", variant.ErrorMessage)
}

func TestBooleanFailure(t *testing.T) {
	evaluationClient, err := evaluation.NewClient(evaluation.WithURL(fliptUrl), evaluation.WithAuthToken(authToken))
	require.NoError(t, err)

	boolean, err := evaluationClient.EvaluateBoolean(context.TODO(), "nonexistent", "someentity", map[string]string{
		"fizz": "buzz",
	})
	require.NoError(t, err)

	assert.Nil(t, boolean.Result)
	assert.Equal(t, "failure", boolean.Status)
	assert.Equal(t, "failed to get flag information default/nonexistent", boolean.ErrorMessage)
}
