package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestEngineVersion(t *testing.T) {
	releases := []release{
		{TagName: "flipt-engine-ffi-v0.21.3"},
		{TagName: "flipt-engine-ffi-v0.21.10"},
		{TagName: "flipt-engine-ffi-v0.22.0", Draft: true},
		{TagName: "flipt-engine-ffi-v0.23.0-rc.1", Prerelease: true},
		{TagName: "flipt-engine-ffi-v0.21.11-rc.1"},
		{TagName: "flipt-engine-ffi-bogus"},
		{TagName: "flipt-engine-wasm-js-v9.0.0"},
		{TagName: "flipt-client-python-v1.4.5"},
	}

	if got := latestEngineVersion("", releases); got != "v0.21.10" {
		t.Errorf("latestEngineVersion() = %q, want v0.21.10", got)
	}

	if got := latestEngineVersion("v0.30.0", releases); got != "v0.30.0" {
		t.Errorf("latestEngineVersion() with higher latest = %q, want v0.30.0", got)
	}

	if got := latestEngineVersion("", nil); got != "" {
		t.Errorf("latestEngineVersion() with no releases = %q, want empty", got)
	}
}

func TestNextPageURL(t *testing.T) {
	tests := map[string]string{
		"": "",
		`<https://api.github.com/repositories/1/releases?page=2>; rel="next", <https://api.github.com/repositories/1/releases?page=5>; rel="last"`:  "https://api.github.com/repositories/1/releases?page=2",
		`<https://api.github.com/repositories/1/releases?page=1>; rel="prev", <https://api.github.com/repositories/1/releases?page=1>; rel="first"`: "",
	}

	for link, want := range tests {
		if got := nextPageURL(link); got != want {
			t.Errorf("nextPageURL(%q) = %q, want %q", link, got, want)
		}
	}
}

func TestGetLatestEngineTag(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret")

	pages := [][]release{
		{{TagName: "flipt-client-go-v1.0.0"}, {TagName: "flipt-engine-ffi-v0.21.3"}},
		{{TagName: "flipt-engine-ffi-v0.21.4"}},
	}

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q, want Bearer secret", got)
		}

		page := 0
		if r.URL.Query().Get("page") == "2" {
			page = 1
		} else {
			w.Header().Set("Link", fmt.Sprintf(`<%s/releases?page=2>; rel="next"`, srv.URL))
		}

		_ = json.NewEncoder(w).Encode(pages[page])
	}))
	defer srv.Close()

	got, err := getLatestEngineTag(context.Background(), srv.Client(), srv.URL+"/releases")
	if err != nil {
		t.Fatal(err)
	}

	if got != "v0.21.4" {
		t.Errorf("getLatestEngineTag() = %q, want v0.21.4", got)
	}
}

func TestGetLatestEngineTagError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"API rate limit exceeded"}`, http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := getLatestEngineTag(context.Background(), srv.Client(), srv.URL+"/releases")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}
