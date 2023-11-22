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

func init() {
	fliptUrl = os.Getenv("FLIPT_URL")
	if fliptUrl == "" {
		panic("set FLIPT_URL")
	}
}

func TestVariant(t *testing.T) {
	evaluationClient, err := evaluation.NewClient(evaluation.WithURL(fliptUrl))
	require.NoError(t, err)

	variant, err := evaluationClient.Variant(context.TODO(), "flag1", "someentity", map[string]string{
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
	evaluationClient, err := evaluation.NewClient(evaluation.WithURL(fliptUrl))
	require.NoError(t, err)

	boolean, err := evaluationClient.Boolean(context.TODO(), "flag_boolean", "someentity", map[string]string{
		"fizz": "buzz",
	})
	require.NoError(t, err)

	assert.Equal(t, "success", boolean.Status)
	assert.Empty(t, boolean.ErrorMessage)
	assert.Equal(t, "flag_boolean", boolean.Result.FlagKey)
	assert.True(t, boolean.Result.Enabled)
	assert.Equal(t, "MATCH_EVALUATION_REASON", boolean.Result.Reason)
}
