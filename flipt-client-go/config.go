package flipt

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// config contains all configuration options for the Flipt client.
type config struct {
	// Environment to fetch flag state from. Defaults to "default".
	Environment string
	// Namespace to fetch flag state from. Defaults to "default".
	Namespace string
	// URL of the upstream Flipt instance. Defaults to "http://localhost:8080".
	URL string
	// RequestTimeout is the timeout for requests to the upstream Flipt instance. Defaults to 30s.
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
