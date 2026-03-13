package flipt

import (
	"net/http"
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config
		wantErr bool
	}{
		{
			name: "valid config (polling)",
			cfg: config{
				Namespace:      "default",
				URL:            "http://localhost:8080/",
				FetchMode:      FetchModePolling,
				UpdateInterval: 2 * time.Second,
				RequestTimeout: 2 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "empty namespace",
			cfg:     config{Namespace: "", URL: "http://localhost:8080", FetchMode: FetchModePolling, UpdateInterval: 2 * time.Second, RequestTimeout: 2 * time.Second},
			wantErr: true,
		},
		{
			name:    "empty url",
			cfg:     config{Namespace: "default", URL: "", FetchMode: FetchModePolling, UpdateInterval: 2 * time.Second, RequestTimeout: 2 * time.Second},
			wantErr: true,
		},
		{
			name:    "update interval too short",
			cfg:     config{Namespace: "default", URL: "http://localhost:8080", FetchMode: FetchModePolling, UpdateInterval: 500 * time.Millisecond, RequestTimeout: 2 * time.Second},
			wantErr: true,
		},
		{
			name:    "request timeout too short",
			cfg:     config{Namespace: "default", URL: "http://localhost:8080", FetchMode: FetchModePolling, UpdateInterval: 2 * time.Second, RequestTimeout: 500 * time.Millisecond},
			wantErr: true,
		},
		{
			name:    "invalid fetch mode",
			cfg:     config{Namespace: "default", URL: "http://localhost:8080", FetchMode: FetchMode("foo"), UpdateInterval: 2 * time.Second, RequestTimeout: 2 * time.Second},
			wantErr: true,
		},
		{
			name:    "valid config (streaming)",
			cfg:     config{Namespace: "default", URL: "http://localhost:8080", FetchMode: FetchModeStreaming},
			wantErr: false,
		},
		{
			name:    "url trailing slash is trimmed",
			cfg:     config{Namespace: "default", URL: "http://localhost:8080/", FetchMode: FetchModeStreaming},
			wantErr: false,
		},
		{
			name:    "http client is set if nil",
			cfg:     config{Namespace: "default", URL: "http://localhost:8080", FetchMode: FetchModeStreaming, HTTPClient: nil},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidate_WithForceAttemptHTTP2_DefaultClient(t *testing.T) {
	t.Parallel()

	cfg := config{
		Namespace:         "default",
		URL:               "http://localhost:8080",
		FetchMode:         FetchModeStreaming,
		ForceAttemptHTTP2: boolPtr(false),
	}

	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	transport, ok := cfg.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", cfg.HTTPClient.Transport)
	}

	if transport.ForceAttemptHTTP2 {
		t.Fatalf("ForceAttemptHTTP2 = true, want false")
	}
}

func TestConfigValidate_WithForceAttemptHTTP2_CustomTransportClone(t *testing.T) {
	t.Parallel()

	customTransport := &http.Transport{ForceAttemptHTTP2: true}
	customClient := &http.Client{Transport: customTransport}

	cfg := config{
		Namespace:         "default",
		URL:               "http://localhost:8080",
		FetchMode:         FetchModeStreaming,
		HTTPClient:        customClient,
		ForceAttemptHTTP2: boolPtr(false),
	}

	if err := cfg.validate(); err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	gotTransport, ok := cfg.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", cfg.HTTPClient.Transport)
	}

	if gotTransport == customTransport {
		t.Fatalf("expected transport clone, got same pointer")
	}

	if gotTransport.ForceAttemptHTTP2 {
		t.Fatalf("ForceAttemptHTTP2 = true, want false")
	}

	if !customTransport.ForceAttemptHTTP2 {
		t.Fatalf("original transport was mutated")
	}
}

func TestConfigValidate_WithForceAttemptHTTP2_InvalidTransport(t *testing.T) {
	t.Parallel()

	cfg := config{
		Namespace:  "default",
		URL:        "http://localhost:8080",
		FetchMode:  FetchModeStreaming,
		HTTPClient: &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) { return nil, nil })},
	}
	WithForceAttemptHTTP2(false)(&cfg)

	if err := cfg.validate(); err == nil {
		t.Fatalf("validate() error = nil, want non-nil")
	}
}

func boolPtr(v bool) *bool {
	return &v
}
