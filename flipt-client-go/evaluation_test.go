package evaluation_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	flipt "go.flipt.io/flipt-client"
)

type EvaluationTestSuite struct {
	suite.Suite
	client    *flipt.EvaluationClient
	fliptUrl  string
	authToken string
}

func (s *EvaluationTestSuite) SetupSuite() {
	opts := []flipt.ClientOption{}

	if os.Getenv("FLIPT_URL") != "" {
		s.fliptUrl = os.Getenv("FLIPT_URL")
		opts = append(opts, flipt.WithURL(s.fliptUrl))
	}

	if os.Getenv("FLIPT_AUTH_TOKEN") != "" {
		s.authToken = os.Getenv("FLIPT_AUTH_TOKEN")
		opts = append(opts, flipt.WithClientTokenAuthentication(s.authToken))
	}

	var err error
	s.client, err = flipt.NewEvaluationClient(context.TODO(), opts...)
	require.NoError(s.T(), err)
}

func (s *EvaluationTestSuite) TearDownSuite() {
	if s.client != nil {
		err := s.client.Close(context.TODO())
		require.NoError(s.T(), err)
	}
}

func TestEvaluationSuite(t *testing.T) {
	suite.Run(t, new(EvaluationTestSuite))
}

func (s *EvaluationTestSuite) TestInvalidAuthentication() {
	_, err := flipt.NewEvaluationClient(context.TODO(), flipt.WithURL(s.fliptUrl), flipt.WithClientTokenAuthentication("invalid"))
	assert.EqualError(s.T(), err, "failed to fetch initial state: unexpected status code: 401")
}

func (s *EvaluationTestSuite) TestVariant() {
	variant, err := s.client.EvaluateVariant(context.TODO(), "flag1", "someentity", map[string]string{
		"fizz": "buzz",
	})

	t := s.T()
	require.NoError(t, err)

	assert.True(t, variant.Match)
	assert.Equal(t, "flag1", variant.FlagKey)
	assert.Equal(t, "MATCH_EVALUATION_REASON", variant.Reason)
	assert.Contains(t, variant.SegmentKeys, "segment1")
}

func (s *EvaluationTestSuite) TestBoolean() {
	boolean, err := s.client.EvaluateBoolean(context.TODO(), "flag_boolean", "someentity", map[string]string{
		"fizz": "buzz",
	})

	t := s.T()
	require.NoError(t, err)

	assert.Equal(t, "flag_boolean", boolean.FlagKey)
	assert.True(t, boolean.Enabled)
	assert.Equal(t, "MATCH_EVALUATION_REASON", boolean.Reason)
}

func (s *EvaluationTestSuite) TestBatch() {
	batch, err := s.client.EvaluateBatch(context.TODO(), []*flipt.EvaluationRequest{
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

	t := s.T()
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

func (s *EvaluationTestSuite) TestListFlags() {
	response, err := s.client.ListFlags(context.TODO())

	t := s.T()
	require.NoError(t, err)

	assert.NotEmpty(t, response)
	assert.Equal(t, 2, len(response))
}

func (s *EvaluationTestSuite) TestVariantFailure() {
	_, err := s.client.EvaluateVariant(context.TODO(), "nonexistent", "someentity", map[string]string{
		"fizz": "buzz",
	})
	assert.EqualError(s.T(), err, "invalid request: failed to get flag information default/nonexistent")
}

func (s *EvaluationTestSuite) TestBooleanFailure() {
	_, err := s.client.EvaluateBoolean(context.TODO(), "nonexistent", "someentity", map[string]string{
		"fizz": "buzz",
	})
	assert.EqualError(s.T(), err, "invalid request: failed to get flag information default/nonexistent")
}
