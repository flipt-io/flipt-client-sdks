package evaluation_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	flipt "go.flipt.io/flipt-client"
)

var (
	evaluationClient *flipt.EvaluationClient
	fliptUrl         string
	authToken        string
)

func init() {
	opts := []flipt.ClientOption{}

	if os.Getenv("FLIPT_URL") != "" {
		opts = append(opts, flipt.WithURL(os.Getenv("FLIPT_URL")))
	}

	if os.Getenv("FLIPT_AUTH_TOKEN") != "" {
		opts = append(opts, flipt.WithClientTokenAuthentication(os.Getenv("FLIPT_AUTH_TOKEN")))
	}

	var err error
	evaluationClient, err = flipt.NewEvaluationClient(context.TODO(), opts...)
	if err != nil {
		panic(err)
	}
}

func TestInvalidAuthentication(t *testing.T) {
	_, err := flipt.NewEvaluationClient(context.TODO(), flipt.WithURL(fliptUrl), flipt.WithClientTokenAuthentication("invalid"))
	assert.EqualError(t, err, "failed to fetch initial state: unexpected status code: 401")
}

func TestVariant(t *testing.T) {
	variant, err := evaluationClient.EvaluateVariant(context.TODO(), "flag1", "someentity", map[string]string{
		"fizz": "buzz",
	})
	require.NoError(t, err)

	assert.True(t, variant.Match)
	assert.Equal(t, "flag1", variant.FlagKey)
	assert.Equal(t, "MATCH_EVALUATION_REASON", variant.Reason)
	assert.Contains(t, variant.SegmentKeys, "segment1")
}

func TestBoolean(t *testing.T) {
	boolean, err := evaluationClient.EvaluateBoolean(context.TODO(), "flag_boolean", "someentity", map[string]string{
		"fizz": "buzz",
	})
	require.NoError(t, err)

	assert.Equal(t, "flag_boolean", boolean.FlagKey)
	assert.True(t, boolean.Enabled)
	assert.Equal(t, "MATCH_EVALUATION_REASON", boolean.Reason)
}

func TestBatch(t *testing.T) {
	batch, err := evaluationClient.EvaluateBatch(context.TODO(), []*flipt.EvaluationRequest{
		{
			FlagKey:  "flag1",
			EntityId: "someentity",
			Context: map[string]string{
				"fizz": "buzz",
			},
		},
		{
			FlagKey:  "flag_boolean",
			EntityId: "someentity",
			Context: map[string]string{
				"fizz": "buzz",
			},
		},
		{
			FlagKey:  "notfound",
			EntityId: "someentity",
			Context: map[string]string{
				"fizz": "buzz",
			},
		},
	})
	require.NoError(t, err)

	assert.Len(t, batch.Responses, 3)

	variant := batch.Responses[0]
	assert.Equal(t, "VARIANT_EVALUATION_RESPONSE_TYPE", variant.Type)
	assert.True(t, variant.VariantEvaluationResponse.Match)
	assert.Equal(t, "flag1", variant.VariantEvaluationResponse.FlagKey)
	assert.Equal(t, "MATCH_EVALUATION_REASON", variant.VariantEvaluationResponse.Reason)
	assert.Contains(t, variant.VariantEvaluationResponse.SegmentKeys, "segment1")

	boolean := batch.Responses[1]
	assert.Equal(t, "BOOLEAN_EVALUATION_RESPONSE_TYPE", boolean.Type)
	assert.Equal(t, "flag_boolean", boolean.BooleanEvaluationResponse.FlagKey)
	assert.True(t, boolean.BooleanEvaluationResponse.Enabled)
	assert.Equal(t, "MATCH_EVALUATION_REASON", boolean.BooleanEvaluationResponse.Reason)

	errorResponse := batch.Responses[2]
	assert.Equal(t, "ERROR_EVALUATION_RESPONSE_TYPE", errorResponse.Type)
	assert.Equal(t, "notfound", errorResponse.ErrorEvaluationResponse.FlagKey)
	assert.Equal(t, "default", errorResponse.ErrorEvaluationResponse.NamespaceKey)
	assert.Equal(t, "NOT_FOUND_ERROR_EVALUATION_REASON", errorResponse.ErrorEvaluationResponse.Reason)
}

func TestListFlags(t *testing.T) {
	response, err := evaluationClient.ListFlags(context.TODO())
	require.NoError(t, err)

	assert.NotEmpty(t, response)
	assert.Equal(t, 2, len(response))
}

func TestVariantFailure(t *testing.T) {
	_, err := evaluationClient.EvaluateVariant(context.TODO(), "nonexistent", "someentity", map[string]string{
		"fizz": "buzz",
	})
	assert.EqualError(t, err, "invalid request: failed to get flag information default/nonexistent")
}

func TestBooleanFailure(t *testing.T) {
	_, err := evaluationClient.EvaluateBoolean(context.TODO(), "nonexistent", "someentity", map[string]string{
		"fizz": "buzz",
	})
	assert.EqualError(t, err, "invalid request: failed to get flag information default/nonexistent")
}
