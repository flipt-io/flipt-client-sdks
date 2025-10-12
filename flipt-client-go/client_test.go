package flipt_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	flipt "go.flipt.io/flipt-client"
)

type ClientTestSuite struct {
	suite.Suite
	client    *flipt.Client
	fliptURL  string
	authToken string
}

func TestClientSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) SetupSuite() {
	opts := []flipt.Option{}

	if os.Getenv("FLIPT_URL") != "" {
		s.fliptURL = os.Getenv("FLIPT_URL")
		opts = append(opts, flipt.WithURL(s.fliptURL))
	}

	if os.Getenv("FLIPT_AUTH_TOKEN") != "" {
		s.authToken = os.Getenv("FLIPT_AUTH_TOKEN")
		opts = append(opts, flipt.WithClientTokenAuthentication(s.authToken))
	}

	// Configure TLS if HTTPS URL is provided
	if s.fliptURL != "" && strings.HasPrefix(s.fliptURL, "https://") {
		caCertPath := os.Getenv("FLIPT_CA_CERT_PATH")
		if caCertPath != "" {
			caCertData, err := os.ReadFile(caCertPath)
			require.NoError(s.T(), err)

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCertData)

			opts = append(opts, flipt.WithTLSConfig(&tls.Config{
				RootCAs: caCertPool,
			}))
		} else {
			// Fallback to insecure for local testing
			opts = append(opts, flipt.WithTLSConfig(&tls.Config{
				InsecureSkipVerify: true,
			}))
		}
	}

	var err error
	s.client, err = flipt.NewClient(context.TODO(), opts...)
	require.NoError(s.T(), err)
}

func (s *ClientTestSuite) TearDownSuite() {
	require.NoError(s.T(), s.client.Close(context.TODO()))
}

func (s *ClientTestSuite) TestInvalidAuthentication() {
	_, err := flipt.NewClient(context.TODO(),
		flipt.WithURL(s.fliptURL),
		flipt.WithClientTokenAuthentication("invalid"),
		flipt.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
	)
	s.EqualError(err, "failed to fetch initial state: unexpected status code: 401")
}

func (s *ClientTestSuite) TestVariant() {
	variant, err := s.client.EvaluateVariant(context.TODO(), &flipt.EvaluationRequest{FlagKey: "flag1", EntityID: "someentity", Context: map[string]string{
		"fizz": "buzz",
	}})
	s.Require().NoError(err)

	s.True(variant.Match)
	s.Equal("flag1", variant.FlagKey)
	s.Equal("MATCH_EVALUATION_REASON", variant.Reason)
	s.Contains(variant.SegmentKeys, "segment1")
}

func (s *ClientTestSuite) TestBoolean() {
	boolean, err := s.client.EvaluateBoolean(context.TODO(), &flipt.EvaluationRequest{FlagKey: "flag_boolean", EntityID: "someentity", Context: map[string]string{
		"fizz": "buzz",
	}})
	s.Require().NoError(err)

	s.Equal("flag_boolean", boolean.FlagKey)
	s.True(boolean.Enabled)
	s.Equal("MATCH_EVALUATION_REASON", boolean.Reason)
	s.Contains(boolean.SegmentKeys, "segment1")
}

func (s *ClientTestSuite) TestBatch() {
	batch, err := s.client.EvaluateBatch(context.TODO(), []*flipt.EvaluationRequest{
		{
			FlagKey:  "flag1",
			EntityID: "someentity",
			Context: map[string]string{
				"fizz": "buzz",
			},
		},
		{
			FlagKey:  "flag_boolean",
			EntityID: "someentity",
			Context: map[string]string{
				"fizz": "buzz",
			},
		},
		{
			FlagKey:  "notfound",
			EntityID: "someentity",
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

func (s *ClientTestSuite) TestListFlags() {
	response, err := s.client.ListFlags(context.TODO())
	s.Require().NoError(err)

	s.NotEmpty(response)
	s.Equal(2, len(response))
}

func (s *ClientTestSuite) TestVariantFailure() {
	_, err := s.client.EvaluateVariant(context.TODO(), &flipt.EvaluationRequest{FlagKey: "nonexistent", EntityID: "someentity", Context: map[string]string{
		"fizz": "buzz",
	}})
	s.EqualError(err, "invalid request: failed to get flag information default/nonexistent")
}

func (s *ClientTestSuite) TestBooleanFailure() {
	_, err := s.client.EvaluateBoolean(context.TODO(), &flipt.EvaluationRequest{FlagKey: "nonexistent", EntityID: "someentity", Context: map[string]string{
		"fizz": "buzz",
	}})
	s.EqualError(err, "invalid request: failed to get flag information default/nonexistent")
}

func (s *ClientTestSuite) TestStreaming() {
	opts := []flipt.Option{
		flipt.WithFetchMode(flipt.FetchModeStreaming),
	}

	if s.fliptURL != "" {
		opts = append(opts, flipt.WithURL(s.fliptURL))
	}

	if s.authToken != "" {
		opts = append(opts, flipt.WithClientTokenAuthentication(s.authToken))
	}

	// Configure TLS if HTTPS URL is provided
	if s.fliptURL != "" && strings.HasPrefix(s.fliptURL, "https://") {
		caCertPath := os.Getenv("FLIPT_CA_CERT_PATH")
		if caCertPath != "" {
			caCertData, err := os.ReadFile(caCertPath)
			require.NoError(s.T(), err)

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCertData)

			opts = append(opts, flipt.WithTLSConfig(&tls.Config{
				RootCAs: caCertPool,
			}))
		} else {
			// Fallback to insecure for local testing
			opts = append(opts, flipt.WithTLSConfig(&tls.Config{
				InsecureSkipVerify: true,
			}))
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.T().Cleanup(cancel)

	streamingClient, err := flipt.NewClient(ctx, opts...)
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		s.Require().NoError(streamingClient.Close(ctx))
	})

	// Wait 1 second as requested before performing evaluation calls
	require.Eventually(s.T(), func() bool {
		return streamingClient.Err() == nil
	}, 2*time.Second, 100*time.Millisecond, "streaming client should be ready")

	// Test variant evaluation with streaming
	variant, err := streamingClient.EvaluateVariant(ctx, &flipt.EvaluationRequest{
		FlagKey:  "flag1",
		EntityID: "someentity",
		Context: map[string]string{
			"fizz": "buzz",
		},
	})
	s.Require().NoError(err)
	s.True(variant.Match)
	s.Equal("flag1", variant.FlagKey)
	s.Equal("MATCH_EVALUATION_REASON", variant.Reason)
	s.Contains(variant.SegmentKeys, "segment1")

	// Test boolean evaluation with streaming
	boolean, err := streamingClient.EvaluateBoolean(ctx, &flipt.EvaluationRequest{
		FlagKey:  "flag_boolean",
		EntityID: "someentity",
		Context: map[string]string{
			"fizz": "buzz",
		},
	})
	s.Require().NoError(err)
	s.Equal("flag_boolean", boolean.FlagKey)
	s.True(boolean.Enabled)
	s.Equal("MATCH_EVALUATION_REASON", boolean.Reason)

	// Test list flags with streaming
	flags, err := streamingClient.ListFlags(ctx)
	s.Require().NoError(err)
	s.NotEmpty(flags)
	s.Equal(2, len(flags))
}

func TestErrorStrategyFallback_InitializationWithInvalidURL(t *testing.T) {
	ctx := context.Background()

	client, err := flipt.NewClient(ctx,
		flipt.WithURL("http://localhost:19999"), // non-existent server
		flipt.WithNamespace("test"),
		flipt.WithErrorStrategy(flipt.ErrorStrategyFallback),
	)

	require.NoError(t, err, "client should initialize with ErrorStrategyFallback even when server is unreachable")
	require.NotNil(t, client, "client should not be nil")

	t.Cleanup(func() {
		_ = client.Close(ctx)
	})

	// The client should have an error stored
	clientErr := client.Err()
	assert.Error(t, clientErr, "client should have stored the fetch error")

	flags, err := client.ListFlags(ctx)
	require.NoError(t, err, "ListFlags should not error with fallback strategy")
	assert.Empty(t, flags, "flags should be empty with empty initial state")
}

func TestErrorStrategyFail_InitializationWithInvalidURL(t *testing.T) {
	ctx := context.Background()

	client, err := flipt.NewClient(ctx,
		flipt.WithURL("http://localhost:19999"), // non-existent server
		flipt.WithNamespace("test"),
		flipt.WithErrorStrategy(flipt.ErrorStrategyFail),
	)

	assert.Error(t, err, "client initialization should fail with ErrorStrategyFail when server is unreachable")
	assert.Nil(t, client, "client should be nil when initialization fails")
}

func TestErrorStrategyDefault_InitializationWithInvalidURL(t *testing.T) {
	ctx := context.Background()

	client, err := flipt.NewClient(ctx,
		flipt.WithURL("http://localhost:19999"), // non-existent server
		flipt.WithNamespace("test"),
	)

	assert.Error(t, err, "client initialization should fail by default when server is unreachable")
	assert.Nil(t, client, "client should be nil when initialization fails")
}

func TestErrorStrategyFallback_EvaluationReturnsError(t *testing.T) {
	ctx := context.Background()

	client, err := flipt.NewClient(ctx,
		flipt.WithURL("http://localhost:19999"), // non-existent server
		flipt.WithNamespace("test"),
		flipt.WithErrorStrategy(flipt.ErrorStrategyFallback),
	)

	require.NoError(t, err, "client should initialize with ErrorStrategyFallback")
	require.NotNil(t, client)

	t.Cleanup(func() {
		_ = client.Close(ctx)
	})

	_, err = client.EvaluateVariant(ctx, &flipt.EvaluationRequest{
		FlagKey:  "test-flag",
		EntityID: "test-entity",
		Context:  map[string]string{},
	})
	assert.Error(t, err, "evaluation should fail when flag doesn't exist in empty state")
	assert.Contains(t, err.Error(), "failed to get flag information", "error should indicate flag not found")
}
