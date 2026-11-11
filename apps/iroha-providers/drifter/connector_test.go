package drifter

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

func TestConnectorFetchSendsBearerAndFollowsCursor(t *testing.T) {
	type request struct {
		query         string
		authorization string
	}
	requests := make([]request, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, request{query: r.URL.Path + "?" + r.URL.RawQuery, authorization: r.Header.Get("Authorization")})
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "" {
			_, _ = w.Write([]byte(`{"version":"1.0","has_more":true,"cursor":"offset:50","books":[{"source_book_id":"book-1","title":"克拉拉与太阳"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"version":"1.0","has_more":false,"books":[{"source_book_id":"book-2","title":"长日将尽"}]}`))
	}))
	defer server.Close()

	client := NewConnector(server.URL, "secret")
	client.Limit = 50
	snapshot, next, err := client.Fetch(context.Background(), connector.Credentials{}, nil)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snapshot.SourceKind != SourceKind || snapshot.SourceInstanceKey != SourceInstanceKey || snapshot.IngestionMode != "full_snapshot" {
		t.Fatalf("snapshot evidence fields = %#v", snapshot)
	}
	if snapshot.ContentType != "application/json" || !strings.Contains(string(snapshot.Body), "克拉拉与太阳") {
		t.Fatalf("snapshot body/content type = %q/%q", snapshot.ContentType, string(snapshot.Body))
	}
	if next == nil || next.Token != "offset:50" {
		t.Fatalf("next cursor = %#v, want token offset:50", next)
	}

	last, lastNext, err := client.Fetch(context.Background(), connector.Credentials{}, next)
	if err != nil {
		t.Fatalf("second Fetch() error = %v", err)
	}
	if lastNext != nil {
		t.Fatalf("second Fetch() cursor = %#v, want nil at end of snapshot", lastNext)
	}
	if !strings.Contains(string(last.Body), "长日将尽") {
		t.Fatalf("second page body = %q", string(last.Body))
	}

	if len(requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(requests))
	}
	if !strings.Contains(requests[0].query, SnapshotPath) || !strings.Contains(requests[0].query, "account=default") || !strings.Contains(requests[0].query, "limit=50") {
		t.Fatalf("first request = %q", requests[0].query)
	}
	if strings.Contains(requests[0].query, "cursor=") {
		t.Fatalf("first request = %q, want no cursor", requests[0].query)
	}
	if !strings.Contains(requests[1].query, "cursor=offset%3A50") {
		t.Fatalf("second request = %q, want the escaped cursor token", requests[1].query)
	}
	for _, got := range requests {
		if got.authorization != "Bearer secret" {
			t.Fatalf("Authorization = %q, want the bearer token", got.authorization)
		}
	}
}

func TestConnectorFetchCredentialsOverrideToken(t *testing.T) {
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"has_more":false,"books":[]}`))
	}))
	defer server.Close()

	client := NewConnector(server.URL, "env-token")
	if _, _, err := client.Fetch(context.Background(), connector.Credentials{Values: map[string]string{"token": "connection-token"}}, nil); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if authorization != "Bearer connection-token" {
		t.Fatalf("Authorization = %q, want the connection credential", authorization)
	}
}

func TestConnectorMapsAuthAndRateLimitFailures(t *testing.T) {
	unauthorized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer unauthorized.Close()

	client := NewConnector(unauthorized.URL, "wrong")
	if _, _, err := client.Fetch(context.Background(), connector.Credentials{}, nil); err == nil {
		t.Fatal("Fetch() returned nil error for 401")
	} else if providerErr, ok := err.(*provider.Error); !ok || providerErr.Kind != provider.ErrorUnavailable || !strings.Contains(providerErr.Error(), "401") {
		t.Fatalf("401 error = %#v", err)
	}

	rateLimited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "9")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer rateLimited.Close()

	client = NewConnector(rateLimited.URL, "secret")
	_, _, err := client.Fetch(context.Background(), connector.Credentials{}, nil)
	providerErr, ok := err.(*provider.Error)
	if !ok || providerErr.Kind != provider.ErrorRateLimited {
		t.Fatalf("429 error = %#v", err)
	}
	if providerErr.RetryAfter == nil || *providerErr.RetryAfter != 9*time.Second {
		t.Fatalf("retry after = %v, want 9s", providerErr.RetryAfter)
	}
}
