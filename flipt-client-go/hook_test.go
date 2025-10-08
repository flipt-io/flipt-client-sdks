package flipt_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	flipt "go.flipt.io/flipt-client"
)

// mockHook is a test implementation of the Hook interface
type mockHook struct {
	mu           sync.Mutex
	beforeCalls  []flipt.BeforeHookData
	afterCalls   []flipt.AfterHookData
	beforeCalled int
	afterCalled  int
}

func (m *mockHook) Before(ctx context.Context, data flipt.BeforeHookData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.beforeCalls = append(m.beforeCalls, data)
	m.beforeCalled++
}

func (m *mockHook) After(ctx context.Context, data flipt.AfterHookData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.afterCalls = append(m.afterCalls, data)
	m.afterCalled++
}

func (m *mockHook) getBeforeCalls() []flipt.BeforeHookData {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]flipt.BeforeHookData{}, m.beforeCalls...)
}

func (m *mockHook) getAfterCalls() []flipt.AfterHookData {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]flipt.AfterHookData{}, m.afterCalls...)
}

func (m *mockHook) getBeforeCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.beforeCalled
}

func (m *mockHook) getAfterCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.afterCalled
}

type HookTestSuite struct {
	suite.Suite
	client    *flipt.Client
	fliptURL  string
	authToken string
	hook      *mockHook
}

func TestHookSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	suite.Run(t, new(HookTestSuite))
}

func (s *HookTestSuite) SetupSuite() {
	s.hook = &mockHook{}
	opts := []flipt.Option{flipt.WithHook(s.hook)}

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

	ctx := context.Background()
	client, err := flipt.NewClient(ctx, opts...)
	require.NoError(s.T(), err)
	s.client = client
}

func (s *HookTestSuite) TearDownSuite() {
	if s.client != nil {
		require.NoError(s.T(), s.client.Close(context.Background()))
	}
}

func (s *HookTestSuite) SetupTest() {
	// Reset hook counters before each test
	s.hook.mu.Lock()
	s.hook.beforeCalls = nil
	s.hook.afterCalls = nil
	s.hook.beforeCalled = 0
	s.hook.afterCalled = 0
	s.hook.mu.Unlock()
}

func (s *HookTestSuite) TestHookVariantEvaluation() {
	ctx := context.Background()

	req := &flipt.EvaluationRequest{
		FlagKey:  "flag1",
		EntityID: "someentity",
		Context: map[string]string{
			"fizz": "buzz",
		},
	}

	resp, err := s.client.EvaluateVariant(ctx, req)
	require.NoError(s.T(), err)

	// Verify hook was called
	require.Equal(s.T(), 1, s.hook.getBeforeCount(), "expected before hook to be called 1 time")
	require.Equal(s.T(), 1, s.hook.getAfterCount(), "expected after hook to be called 1 time")

	// Verify before hook data
	beforeCalls := s.hook.getBeforeCalls()
	require.Len(s.T(), beforeCalls, 1)
	require.Equal(s.T(), "flag1", beforeCalls[0].FlagKey)

	// Verify after hook data
	afterCalls := s.hook.getAfterCalls()
	require.Len(s.T(), afterCalls, 1)
	afterData := afterCalls[0]
	require.Equal(s.T(), resp.FlagKey, afterData.FlagKey)
	require.Equal(s.T(), "variant", afterData.FlagType)
	require.Equal(s.T(), resp.VariantKey, afterData.Value)
	require.Equal(s.T(), resp.Reason, afterData.Reason)
}

func (s *HookTestSuite) TestHookBooleanEvaluation() {
	ctx := context.Background()

	req := &flipt.EvaluationRequest{
		FlagKey:  "flag_boolean",
		EntityID: "someentity",
		Context: map[string]string{
			"fizz": "buzz",
		},
	}

	resp, err := s.client.EvaluateBoolean(ctx, req)
	require.NoError(s.T(), err)

	// Verify hook was called
	require.Equal(s.T(), 1, s.hook.getBeforeCount(), "expected before hook to be called 1 time")
	require.Equal(s.T(), 1, s.hook.getAfterCount(), "expected after hook to be called 1 time")

	// Verify after hook data
	afterCalls := s.hook.getAfterCalls()
	require.Len(s.T(), afterCalls, 1)
	afterData := afterCalls[0]
	require.Equal(s.T(), resp.FlagKey, afterData.FlagKey)
	require.Equal(s.T(), "boolean", afterData.FlagType)
}

func (s *HookTestSuite) TestHookBatchEvaluation() {
	ctx := context.Background()

	requests := []*flipt.EvaluationRequest{
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
	}

	resp, err := s.client.EvaluateBatch(ctx, requests)
	require.NoError(s.T(), err)

	// Verify before hook was called for each request
	require.Equal(s.T(), 2, s.hook.getBeforeCount(), "expected before hook to be called 2 times")

	// Verify after hook was called for each successful response
	expectedAfterCalls := len(resp.Responses)
	require.Equal(s.T(), expectedAfterCalls, s.hook.getAfterCount())

	// Verify before hook data
	beforeCalls := s.hook.getBeforeCalls()
	require.Len(s.T(), beforeCalls, 2)
	require.Equal(s.T(), "flag1", beforeCalls[0].FlagKey)
	require.Equal(s.T(), "flag_boolean", beforeCalls[1].FlagKey)
}
