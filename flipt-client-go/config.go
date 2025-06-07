package flipt

import (
	"net/http"
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
