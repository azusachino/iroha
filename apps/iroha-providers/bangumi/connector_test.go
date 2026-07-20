package bangumi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

func TestConnectorFetchSendsUserAgentAndPaginates(t *testing.T) {
	var gotPath, gotUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		gotUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"total":2,"data":[{"subject_type":2,"type":3,"subject":{"id":20,"name":"NARUTO"}}]}`))
	}))
	defer server.Close()

	client := NewConnector("mikufan2039", "secret")
	client.Endpoint = server.URL
	client.Limit = 1
	snapshot, next, err := client.Fetch(context.Background(), connector.Credentials{}, nil)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snapshot.SourceKind != SourceKind || next == nil || next.Page != 1 {
		t.Fatalf("snapshot/cursor = %#v/%#v", snapshot, next)
	}
	if !strings.Contains(gotPath, "limit=1") || !strings.Contains(gotPath, "offset=0") || gotUserAgent == "" {
		t.Fatalf("request path/user-agent = %q/%q", gotPath, gotUserAgent)
	}
}

func TestConnectorMapsRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "9")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewConnector("mikufan2039", "")
	client.Endpoint = server.URL
	_, _, err := client.Fetch(context.Background(), connector.Credentials{}, nil)
	if err == nil {
		t.Fatal("Fetch() returned nil error for 429")
	}
	providerErr, ok := err.(*provider.Error)
	if !ok || providerErr.Kind != provider.ErrorRateLimited || !strings.Contains(providerErr.Error(), "Retry-After=9") {
		t.Fatalf("rate limit error = %#v", err)
	}
	if providerErr.RetryAfter == nil || *providerErr.RetryAfter != 9*time.Second {
		t.Fatalf("retry after = %v, want 9s", providerErr.RetryAfter)
	}
}
