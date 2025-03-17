package evaluation_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	flipt "go.flipt.io/flipt-client"
)

type EvaluationTestSuite struct {
	suite.Suite
	client    *flipt.EvaluationClient
	fliptURL  string
	authToken string
}

func TestEvaluationSuite(t *testing.T) {
	suite.Run(t, new(EvaluationTestSuite))
}

func (s *EvaluationTestSuite) SetupSuite() {
	opts := []flipt.ClientOption{}

	if os.Getenv("FLIPT_URL") != "" {
		s.fliptURL = os.Getenv("FLIPT_URL")
		opts = append(opts, flipt.WithURL(s.fliptURL))
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
	s.client.Close(context.TODO())
}

func (s *EvaluationTestSuite) TestInvalidAuthentication() {
	_, err := flipt.NewEvaluationClient(context.TODO(),
		flipt.WithURL(s.fliptURL),
		flipt.WithClientTokenAuthentication("invalid"))
	s.EqualError(err, "failed to fetch initial state: unexpected status code: 401")
}

func (s *EvaluationTestSuite) TestVariant() {
	variant, err := s.client.EvaluateVariant(context.TODO(), "flag1", "someentity", map[string]string{
		"fizz": "buzz",
	})
	s.Require().NoError(err)

	s.True(variant.Match)
	s.Equal("flag1", variant.FlagKey)
	s.Equal("MATCH_EVALUATION_REASON", variant.Reason)
	s.Contains(variant.SegmentKeys, "segment1")
}

func (s *EvaluationTestSuite) TestBoolean() {
	boolean, err := s.client.EvaluateBoolean(context.TODO(), "flag_boolean", "someentity", map[string]string{
		"fizz": "buzz",
	})
	s.Require().NoError(err)

	s.Equal("flag_boolean", boolean.FlagKey)
	s.True(boolean.Enabled)
	s.Equal("MATCH_EVALUATION_REASON", boolean.Reason)
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
	s.Require().NoError(err)

	s.Len(batch.Responses, 3)

	variant := batch.Responses[0]
	s.Require().Equal("VARIANT_EVALUATION_RESPONSE_TYPE", variant.Type)
	s.True(variant.VariantEvaluationResponse.Match)
	s.Equal("flag1", variant.VariantEvaluationResponse.FlagKey)
	s.Equal("MATCH_EVALUATION_REASON", variant.VariantEvaluationResponse.Reason)
	s.Contains(variant.VariantEvaluationResponse.SegmentKeys, "segment1")

	boolean := batch.Responses[1]
	s.Equal("BOOLEAN_EVALUATION_RESPONSE_TYPE", boolean.Type)
	s.Equal("flag_boolean", boolean.BooleanEvaluationResponse.FlagKey)
	s.True(boolean.BooleanEvaluationResponse.Enabled)
	s.Equal("MATCH_EVALUATION_REASON", boolean.BooleanEvaluationResponse.Reason)

	errorResponse := batch.Responses[2]
	s.Equal("ERROR_EVALUATION_RESPONSE_TYPE", errorResponse.Type)
	s.Equal("notfound", errorResponse.ErrorEvaluationResponse.FlagKey)
	s.Equal("default", errorResponse.ErrorEvaluationResponse.NamespaceKey)
	s.Equal("NOT_FOUND_ERROR_EVALUATION_REASON", errorResponse.ErrorEvaluationResponse.Reason)
}

func (s *EvaluationTestSuite) TestListFlags() {
	response, err := s.client.ListFlags(context.TODO())
	s.Require().NoError(err)

	s.NotEmpty(response)
	s.Equal(2, len(response))
}

func (s *EvaluationTestSuite) TestVariantFailure() {
	_, err := s.client.EvaluateVariant(context.TODO(), "nonexistent", "someentity", map[string]string{
		"fizz": "buzz",
	})
	s.EqualError(err, "invalid request: failed to get flag information default/nonexistent")
}

func (s *EvaluationTestSuite) TestBooleanFailure() {
	_, err := s.client.EvaluateBoolean(context.TODO(), "nonexistent", "someentity", map[string]string{
		"fizz": "buzz",
	})
	s.EqualError(err, "invalid request: failed to get flag information default/nonexistent")
}
