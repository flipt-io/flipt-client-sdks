package flipt

import (
	"context"
	"net/http"
	"testing"
)

func TestClientNewRequest_LegacyHeadersAuto(t *testing.T) {
	t.Parallel()

	client := &Client{
		cfg: config{
			Environment:       "prod",
			LegacyHeadersMode: legacyHeadersModeAuto,
		},
	}

	v1Req, err := client.newRequest(context.Background(), "http://localhost:8080/internal/v1/evaluation/snapshot/namespace/default")
	if err != nil {
		t.Fatalf("newRequest() error = %v", err)
	}

	if got := v1Req.Header.Get("x-flipt-accept-server-version"); got == "" {
		t.Fatalf("expected x-flipt-accept-server-version on non-v2 endpoint")
	}

	if got := v1Req.Header.Get("x-flipt-environment"); got != "prod" {
		t.Fatalf("x-flipt-environment = %q, want %q", got, "prod")
	}

	v2Req, err := client.newRequest(context.Background(), "http://localhost:8080/client/v2/environments/prod/namespaces/default/stream")
	if err != nil {
		t.Fatalf("newRequest() error = %v", err)
	}

	if got := v2Req.Header.Get("x-flipt-accept-server-version"); got != "" {
		t.Fatalf("expected no x-flipt-accept-server-version on v2 endpoint, got %q", got)
	}

	if got := v2Req.Header.Get("x-flipt-environment"); got != "" {
		t.Fatalf("expected no x-flipt-environment on v2 endpoint, got %q", got)
	}
}

func TestClientNewRequest_LegacyHeadersOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		mode              legacyHeadersMode
		url               string
		wantLegacyHeaders bool
	}{
		{
			name:              "enabled override keeps headers for v2",
			mode:              legacyHeadersModeEnabled,
			url:               "http://localhost:8080/client/v2/environments/prod/namespaces/default/stream",
			wantLegacyHeaders: true,
		},
		{
			name:              "disabled override removes headers for v1",
			mode:              legacyHeadersModeDisabled,
			url:               "http://localhost:8080/internal/v1/evaluation/snapshot/namespace/default",
			wantLegacyHeaders: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				cfg: config{
					Environment:       "prod",
					LegacyHeadersMode: tt.mode,
				},
			}

			req, err := client.newRequest(context.Background(), tt.url)
			if err != nil {
				t.Fatalf("newRequest() error = %v", err)
			}

			gotServerVersion := req.Header.Get("x-flipt-accept-server-version") != ""
			gotEnvironment := req.Header.Get("x-flipt-environment") != ""

			if gotServerVersion != tt.wantLegacyHeaders {
				t.Fatalf("x-flipt-accept-server-version present = %v, want %v", gotServerVersion, tt.wantLegacyHeaders)
			}

			if gotEnvironment != tt.wantLegacyHeaders {
				t.Fatalf("x-flipt-environment present = %v, want %v", gotEnvironment, tt.wantLegacyHeaders)
			}
		})
	}
}

func TestWithLegacyHeadersOption(t *testing.T) {
	t.Parallel()

	cfg := config{}
	WithLegacyHeaders(true)(&cfg)
	if cfg.LegacyHeadersMode != legacyHeadersModeEnabled {
		t.Fatalf("LegacyHeadersMode = %v, want %v", cfg.LegacyHeadersMode, legacyHeadersModeEnabled)
	}

	WithLegacyHeaders(false)(&cfg)
	if cfg.LegacyHeadersMode != legacyHeadersModeDisabled {
		t.Fatalf("LegacyHeadersMode = %v, want %v", cfg.LegacyHeadersMode, legacyHeadersModeDisabled)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (r roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
