package flipt

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
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

// WithTlsConfig sets the TLS configuration for the client using the standard library tls.Config.
// This provides maximum flexibility for configuring TLS settings including custom CAs,
// mutual TLS authentication, and certificate verification options.
func WithTlsConfig(tlsConfig *tls.Config) Option {
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
				} else {
					fmt.Println("Warning: HTTPClient.Transport is not of type *http.Transport. Falling back to default transport.")
					transport = defaultHTTPClient.Transport.(*http.Transport).Clone()
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
}

// validate validates the configuration and sets defaults.
func (c *config) validate() error {
	if c.HTTPClient == nil {
		c.HTTPClient = defaultHTTPClient
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
