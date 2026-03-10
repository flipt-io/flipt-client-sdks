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
			Environment: "prod",
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

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (r roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return r(req)
}
