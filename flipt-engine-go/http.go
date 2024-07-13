package flipt_engine_go

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

const basePath = "internal/v1/evaluation/snapshot/namespace/"

type HTTPParser struct {
	client       *http.Client
	host         string
	fliptVersion string
}

func NewHTTPParser(host, fliptVersion string, timeout time.Duration) *HTTPParser {
	client := http.DefaultClient
	client.Timeout = timeout

	return &HTTPParser{
		client:       client,
		host:         host,
		fliptVersion: fliptVersion,
	}
}

func (r *HTTPParser) Do(ctx context.Context, namespace string) ([]byte, error) {
	u := url.URL{
		Scheme: "http",
		Host:   r.host,
		Path:   path.Join(basePath, namespace),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil) //nolint:gocritic // Initial linter integration.
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Flipt-Accept-Server-Version", r.fliptVersion)

	response, err := r.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}
	defer func() {
		if response != nil {
			response.Body.Close()
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read all: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status is not success, code: %d, body: %s", response.StatusCode, string(body))
	}

	return body, nil
}

func (r *HTTPParser) Parse(payload []byte) (*Document, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}
	var doc Document
	err := json.Unmarshal(payload, &doc)
	if err != nil {
		return nil, err
	}

	return &doc, nil
}
