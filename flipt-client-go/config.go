package flipt

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type legacyHeadersMode uint8

const (
	legacyHeadersModeAuto legacyHeadersMode = iota
	legacyHeadersModeEnabled
	legacyHeadersModeDisabled
)

// Option configures the Flipt client via the provided config struct.
type Option func(*config)

// WithEnvironment sets the environment for the client.
func WithEnvironment(environment string) Option {
	return func(cfg *config) {
		cfg.Environment = environment
	}
}

// WithNamespace sets the namespace for the client.
func WithNamespace(namespace string) Option {
	return func(cfg *config) {
		cfg.Namespace = namespace
	}
}

// WithURL sets the URL for the client.
func WithURL(url string) Option {
	return func(cfg *config) {
		cfg.URL = url
	}
}

// WithHTTPClient sets the HTTP client for the client.
// Note: if used in conjunction with WithRequestTimeout the request timeout on the HTTP client will be overridden.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(cfg *config) {
		cfg.HTTPClient = httpClient
	}
}

// WithLegacyHeaders controls legacy Flipt request headers:
// x-flipt-accept-server-version and x-flipt-environment.
// When unset, behavior is automatic:
// - non-v2 client paths: enabled
// - /client/v2 paths: disabled
func WithLegacyHeaders(enabled bool) Option {
	return func(cfg *config) {
		if enabled {
			cfg.LegacyHeadersMode = legacyHeadersModeEnabled
			return
		}

		cfg.LegacyHeadersMode = legacyHeadersModeDisabled
	}
}

// WithForceAttemptHTTP2 sets the ForceAttemptHTTP2 option on the client's http.Transport
// without requiring the caller to replace the entire HTTP client.
func WithForceAttemptHTTP2(forceAttemptHTTP2 bool) Option {
	return func(cfg *config) {
		cfg.ForceAttemptHTTP2 = &forceAttemptHTTP2
	}
}

// WithReference sets the reference for the client.
func WithReference(ref string) Option {
	return func(cfg *config) {
		cfg.Reference = ref
	}
}

// WithUpdateInterval sets the update interval for the client.
func WithUpdateInterval(updateInterval time.Duration) Option {
	return func(cfg *config) {
		cfg.UpdateInterval = updateInterval
	}
}

// WithClientTokenAuthentication sets client token authentication.
func WithClientTokenAuthentication(token string) Option {
	return func(cfg *config) {
		cfg.Authentication = clientTokenAuthentication{Token: token}
	}
}

// WithJWTAuthentication sets JWT authentication.
func WithJWTAuthentication(token string) Option {
	return func(cfg *config) {
		cfg.Authentication = jwtAuthentication{Token: token}
	}
}

// WithFetchMode sets the fetch mode for the client.
func WithFetchMode(fetchMode FetchMode) Option {
	return func(cfg *config) {
		cfg.FetchMode = fetchMode
	}
}

// WithErrorStrategy sets the error strategy for the client.
func WithErrorStrategy(errorStrategy ErrorStrategy) Option {
	return func(cfg *config) {
		cfg.ErrorStrategy = errorStrategy
	}
}

// WithRequestTimeout sets the request timeout for the client.
// Note: this is only used for polling mode.
// Note: setting this will override the HTTP client timeout.
func WithRequestTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.RequestTimeout = timeout
	}
}

// WithHook sets the hook for the client.
func WithHook(hook Hook) Option {
	return func(cfg *config) {
		cfg.Hook = hook
	}
}

// WithTLSConfig sets the TLS configuration for the client using the standard library tls.Config.
// This provides maximum flexibility for configuring TLS settings including custom CAs,
// mutual TLS authentication, and certificate verification options.
// Note: if used with WithHTTPClient, this should be called after setting the HTTP client.
func WithTLSConfig(tlsConfig *tls.Config) Option {
	return func(cfg *config) {
		if tlsConfig == nil {
			return
		}

		// Create a new HTTP client based on the one provided or the default one
		var (
			transport = defaultHTTPClient.Transport.(*http.Transport).Clone()
			timeout   = defaultHTTPClient.Timeout
		)

		if cfg.HTTPClient != nil {
			timeout = cfg.HTTPClient.Timeout
			if cfg.HTTPClient.Transport != nil {
				if t, ok := cfg.HTTPClient.Transport.(*http.Transport); ok {
					transport = t.Clone()
				}
			}
		}

		// Apply the provided TLS config
		transport.TLSClientConfig = tlsConfig

		cfg.HTTPClient = &http.Client{
			Transport: transport,
			Timeout:   timeout,
		}
	}
}

// defaultHTTPClient is the default HTTP client used by the client.
// It is used to make requests to the upstream Flipt instance and is configured to be best compatible with streaming mode.
var defaultHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 5 * time.Second, // More aggressive TCP keepalive
		}).DialContext,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		// Enable HTTP/2 ping frames and other settings
		Proxy:           http.ProxyFromEnvironment,
		ReadBufferSize:  32 * 1024,
		WriteBufferSize: 32 * 1024,
		// HTTP/2 specific settings
		MaxResponseHeaderBytes: 4 * 1024, // 4KB
		DisableCompression:     false,    // Enable compression
	},
}

// config contains all configuration options for the Flipt client.
type config struct {
	// Environment to fetch flag state from. Defaults to "default".
	Environment string
	// Namespace to fetch flag state from. Defaults to "default".
	Namespace string
	// URL of the upstream Flipt instance. Defaults to "http://localhost:8080".
	URL string
	// RequestTimeout is the timeout for requests to the upstream Flipt instance. Defaults to no timeout.
	RequestTimeout time.Duration
	// UpdateInterval is how often to fetch new flag state. Defaults to 120s.
	UpdateInterval time.Duration
	// Authentication strategy (e.g., client token, JWT). Optional.
	Authentication any
	// Reference to use when fetching flag state. Optional.
	Reference string
	// FetchMode determines how flag state is fetched (polling or streaming). Defaults to polling.
	FetchMode FetchMode
	// ErrorStrategy determines how errors are handled (fail or fallback). Defaults to fail.
	ErrorStrategy ErrorStrategy
	// HTTPClient to use for requests. If nil, a default client is used.
	HTTPClient *http.Client
	// Hook for before and after evaluation callbacks. Optional.
	Hook Hook
	// LegacyHeadersMode controls legacy request headers behavior.
	// Auto mode enables legacy headers for non-v2 paths only.
	LegacyHeadersMode legacyHeadersMode
	// ForceAttemptHTTP2 optionally overrides the transport's HTTP/2 negotiation behavior.
	ForceAttemptHTTP2 *bool
}

// validate validates the configuration and sets defaults.
func (c *config) validate() error {
	if c.HTTPClient == nil {
		c.HTTPClient = defaultHTTPClient
	}

	if err := c.applyForceAttemptHTTP2(); err != nil {
		return err
	}

	if c.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	if c.URL == "" {
		return fmt.Errorf("baseURL cannot be empty")
	}

	if c.FetchMode == FetchModePolling {
		if c.UpdateInterval < 1*time.Second {
			return fmt.Errorf("update interval must be greater than 0")
		}

		if c.RequestTimeout != 0 && c.RequestTimeout < 1*time.Second {
			return fmt.Errorf("request timeout must be greater than or equal to 0")
		}
	}

	// Store the base URL without trailing slash
	c.URL = strings.TrimRight(c.URL, "/")

	// Set timeout only for polling mode
	if c.FetchMode == FetchModePolling {
		c.HTTPClient.Timeout = c.RequestTimeout
	}

	// Validate fetch mode
	if c.FetchMode != FetchModePolling && c.FetchMode != FetchModeStreaming {
		return fmt.Errorf("invalid fetch mode: %s", c.FetchMode)
	}

	return nil
}

func (c *config) applyForceAttemptHTTP2() error {
	if c.ForceAttemptHTTP2 == nil {
		return nil
	}

	httpClient := *c.HTTPClient

	switch transport := c.HTTPClient.Transport.(type) {
	case nil:
		defaultTransport, ok := defaultHTTPClient.Transport.(*http.Transport)
		if !ok {
			return fmt.Errorf("default transport is not *http.Transport")
		}

		cloned := defaultTransport.Clone()
		cloned.ForceAttemptHTTP2 = *c.ForceAttemptHTTP2
		httpClient.Transport = cloned
	case *http.Transport:
		cloned := transport.Clone()
		cloned.ForceAttemptHTTP2 = *c.ForceAttemptHTTP2
		httpClient.Transport = cloned
	default:
		return fmt.Errorf("http client transport must be *http.Transport when using WithForceAttemptHTTP2")
	}

	c.HTTPClient = &httpClient
	return nil
}
