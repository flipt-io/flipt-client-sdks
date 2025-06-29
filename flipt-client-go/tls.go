package flipt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TlsConfig provides configuration for TLS connections to Flipt servers.
// This is useful when connecting to servers with self-signed certificates,
// using custom Certificate Authorities (CAs), implementing mutual TLS
// authentication, or testing with insecure connections (development only).
type TlsConfig struct {
	// caCertData is the raw CA certificate content (PEM format).
	caCertData []byte

	// insecureSkipVerify controls whether the client verifies the server's
	// certificate chain and host name. If true, TLS accepts any certificate
	// presented by the server and any host name in that certificate.
	// WARNING: Only use this in development environments!
	insecureSkipVerify bool

	// clientCertData is the raw client certificate content for mutual TLS (PEM format).
	clientCertData []byte

	// clientKeyData is the raw client private key content for mutual TLS (PEM format).
	clientKeyData []byte
}

// Insecure creates a TLS configuration that skips certificate verification.
// WARNING: Only use this in development environments!
func InsecureTlsConfig() *TlsConfig {
	return &TlsConfig{
		insecureSkipVerify: true,
	}
}

// WithCaCert creates a TLS configuration with CA certificate data.
func WithCaCert(caCertData io.Reader) (*TlsConfig, error) {
	caCertDataBytes, err := io.ReadAll(caCertData)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate data: %w", err)
	}

	return &TlsConfig{
		caCertData: caCertDataBytes,
	}, nil
}

// WithMutualTls creates a TLS configuration for mutual TLS using certificate data.
func WithMutualTls(clientCertData, clientKeyData io.Reader) (*TlsConfig, error) {
	clientCertDataBytes, err := io.ReadAll(clientCertData)
	if err != nil {
		return nil, fmt.Errorf("failed to read client certificate data: %w", err)
	}

	clientKeyDataBytes, err := io.ReadAll(clientKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to read client key data: %w", err)
	}

	return &TlsConfig{
		clientCertData: clientCertDataBytes,
		clientKeyData:  clientKeyDataBytes,
	}, nil
}

// buildTLSConfig creates a tls.Config based on the TlsConfig settings.
func (tc *TlsConfig) buildTLSConfig() (*tls.Config, error) {
	if tc == nil {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: tc.insecureSkipVerify,
	}

	// Configure custom CA certificate
	if err := tc.configureCA(tlsConfig); err != nil {
		return nil, fmt.Errorf("failed to configure CA certificate: %w", err)
	}

	// Configure client certificate for mutual TLS
	if err := tc.configureClientCert(tlsConfig); err != nil {
		return nil, fmt.Errorf("failed to configure client certificate: %w", err)
	}

	return tlsConfig, nil
}

// configureCA configures the CA certificate for the TLS connection.
func (tc *TlsConfig) configureCA(tlsConfig *tls.Config) error {
	var caCertPEM []byte

	if len(tc.caCertData) > 0 {
		caCertPEM = tc.caCertData
	}

	if len(caCertPEM) > 0 {
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCertPEM) {
			return fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return nil
}

// configureClientCert configures the client certificate for mutual TLS.
func (tc *TlsConfig) configureClientCert(tlsConfig *tls.Config) error {
	var clientCertPEM, clientKeyPEM []byte

	if len(tc.clientCertData) > 0 && len(tc.clientKeyData) > 0 {
		clientCertPEM = tc.clientCertData
		clientKeyPEM = tc.clientKeyData
	}

	if len(clientCertPEM) > 0 && len(clientKeyPEM) > 0 {
		clientCert, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
		if err != nil {
			return fmt.Errorf("failed to parse client certificate and key: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	return nil
}

// WithTlsConfig sets the TLS configuration for the client.
// This creates a new HTTP client with the specified TLS settings.
func WithTlsConfig(tlsConfig *TlsConfig) Option {
	return func(cfg *config) {
		if tlsConfig == nil {
			return
		}

		// Create a new HTTP client based on the one provided
		var (
			transport *http.Transport = defaultHTTPClient.Transport.(*http.Transport).Clone()
			timeout   time.Duration   = defaultHTTPClient.Timeout
		)

		if cfg.HTTPClient.Transport != nil {
			transport = cfg.HTTPClient.Transport.(*http.Transport).Clone()
			timeout = cfg.HTTPClient.Timeout
		}

		// Build and apply TLS config
		if tlsConf, err := tlsConfig.buildTLSConfig(); err == nil {
			transport.TLSClientConfig = tlsConf
		}

		cfg.HTTPClient = &http.Client{
			Transport: transport,
			Timeout:   timeout,
		}
	}
}
