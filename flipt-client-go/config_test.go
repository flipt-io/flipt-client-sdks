package flipt

import (
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
