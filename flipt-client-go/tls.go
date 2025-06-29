package flipt

import (
	"crypto/tls"
	"net/http"
	"time"
)

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
			transport *http.Transport = defaultHTTPClient.Transport.(*http.Transport).Clone()
			timeout   time.Duration   = defaultHTTPClient.Timeout
		)

		if cfg.HTTPClient != nil {
			timeout = cfg.HTTPClient.Timeout
			if cfg.HTTPClient.Transport != nil {
				transport = cfg.HTTPClient.Transport.(*http.Transport).Clone()
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
