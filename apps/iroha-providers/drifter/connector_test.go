package drifter

import (
	"context"
	"errors"
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
			_, _ = w.Write([]byte(`{"version":"1.0","source_identity":"weread:23849102","watermark":"2026-11-11T10:00:00Z","has_more":true,"cursor":"offset:50","books":[{"source_book_id":"book-1","title":"克拉拉与太阳"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"version":"1.0","source_identity":"weread:23849102","watermark":"2026-11-11T10:00:00Z","has_more":false,"books":[{"source_book_id":"book-2","title":"长日将尽"}]}`))
	}))
	defer server.Close()

	client := NewConnector(server.URL, "secret")
	client.Limit = 50
	snapshot, next, err := client.Fetch(context.Background(), connector.Credentials{}, nil)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snapshot.SourceKind != SourceKind || snapshot.SourceInstanceKey != SourceInstanceKey || snapshot.IngestionMode != IngestionModeSnapshot {
		t.Fatalf("snapshot evidence fields = %#v", snapshot)
	}
	if snapshot.OrderingBasis != "2026-11-11T10:00:00Z" {
		t.Fatalf("snapshot ordering basis = %q, want the page watermark", snapshot.OrderingBasis)
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
	// The gateway names its own account: an account parameter would re-imply a
	// server default the owner never chose.
	if !strings.Contains(requests[0].query, SnapshotPath) || strings.Contains(requests[0].query, "account=") || !strings.Contains(requests[0].query, "limit=50") {
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

func TestNewConnectorDefaultsRespectGatewayPacing(t *testing.T) {
	client := NewConnector("", "")
	if client.Limit != 10 {
		t.Fatalf("default page limit = %d, want 10", client.Limit)
	}
	if client.Client == nil || client.Client.Timeout != 60*time.Second {
		t.Fatalf("default client timeout = %v, want 60s", client.Client)
	}
	if client.BaseURL != DefaultBaseURL {
		t.Fatalf("default base URL = %q, want %q", client.BaseURL, DefaultBaseURL)
	}
}

func TestConnectorFetchCredentialsOverrideToken(t *testing.T) {
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"version":"1.0","has_more":false,"books":[]}`))
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

func TestConnectorFetchChangesSendsSinceAndCursor(t *testing.T) {
	var request string
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request = r.URL.Path + "?" + r.URL.RawQuery
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"1.0","source_identity":"weread:23849102","watermark":"2026-11-11T15:00:00Z","has_more":true,"cursor":"offset:100","books":[{"source_book_id":"book-1","title":"克拉拉与太阳","progress_percent":40}]}`))
	}))
	defer server.Close()

	since := time.Date(2026, time.November, 11, 12, 0, 0, 0, time.UTC)
	client := NewConnector(server.URL, "secret")
	page, err := client.FetchChanges(context.Background(), connector.Credentials{}, since, &connector.Cursor{Token: "offset:0"})
	if err != nil {
		t.Fatalf("FetchChanges() error = %v", err)
	}
	if !strings.Contains(request, ChangesPath) {
		t.Fatalf("request = %q, want the changes path", request)
	}
	if !strings.Contains(request, "since=2026-11-11T12%3A00%3A00Z") {
		t.Fatalf("request = %q, want the RFC3339 watermark", request)
	}
	if !strings.Contains(request, "limit=10") {
		t.Fatalf("request = %q, want the shared page size", request)
	}
	if !strings.Contains(request, "cursor=offset%3A0") {
		t.Fatalf("request = %q, want the escaped cursor token", request)
	}
	if authorization != "Bearer secret" {
		t.Fatalf("Authorization = %q, want the bearer token", authorization)
	}
	if page.Watermark != "2026-11-11T15:00:00Z" {
		t.Fatalf("watermark = %q, want the page watermark", page.Watermark)
	}
	if page.Cursor == nil || page.Cursor.Token != "offset:100" {
		t.Fatalf("next cursor = %#v, want token offset:100", page.Cursor)
	}
	if page.Snapshot.IngestionMode != IngestionModeChanges {
		t.Fatalf("ingestion mode = %q, want %q", page.Snapshot.IngestionMode, IngestionModeChanges)
	}
	if page.Snapshot.OrderingBasis != "2026-11-11T15:00:00Z" {
		t.Fatalf("ordering basis = %q, want the page watermark", page.Snapshot.OrderingBasis)
	}
	if page.Snapshot.Filename != changesFilename {
		t.Fatalf("filename = %q, want %q", page.Snapshot.Filename, changesFilename)
	}
	if !strings.Contains(string(page.Snapshot.Body), "克拉拉与太阳") {
		t.Fatalf("changes body = %q", string(page.Snapshot.Body))
	}
}

func TestConnectorFetchChangesRequiresWatermark(t *testing.T) {
	client := NewConnector("http://127.0.0.1:1", "secret")
	if _, err := client.FetchChanges(context.Background(), connector.Credentials{}, time.Time{}, nil); err == nil {
		t.Fatal("FetchChanges() accepted a zero watermark")
	}
}

func TestConnectorFetchChangesMapsExpiredHorizon(t *testing.T) {
	since := time.Date(2026, time.November, 11, 12, 0, 0, 0, time.UTC)

	gone := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte(`{"error":"horizon_exceeded","snapshot_required":true}`))
	}))
	defer gone.Close()

	if _, err := NewConnector(gone.URL, "secret").FetchChanges(context.Background(), connector.Credentials{}, since, nil); !errors.Is(err, ErrSnapshotRequired) {
		t.Fatalf("410 error = %v, want ErrSnapshotRequired", err)
	}

	// A gateway may also say so in the body with a 200; the caller must not
	// treat an unusable feed page as usable evidence.
	advisory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"snapshot_required":true,"watermark":"2026-11-11T12:00:00Z"}`))
	}))
	defer advisory.Close()

	if _, err := NewConnector(advisory.URL, "secret").FetchChanges(context.Background(), connector.Credentials{}, since, nil); !errors.Is(err, ErrSnapshotRequired) {
		t.Fatalf("snapshot_required body error = %v, want ErrSnapshotRequired", err)
	}
}

func TestConnectorFetchChangesMapsRateLimit(t *testing.T) {
	rateLimited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "9")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer rateLimited.Close()

	client := NewConnector(rateLimited.URL, "secret")
	_, err := client.FetchChanges(context.Background(), connector.Credentials{}, time.Now().UTC(), nil)
	providerErr, ok := err.(*provider.Error)
	if !ok || providerErr.Kind != provider.ErrorRateLimited {
		t.Fatalf("429 error = %#v", err)
	}
	if providerErr.RetryAfter == nil || *providerErr.RetryAfter != 9*time.Second {
		t.Fatalf("retry after = %v, want 9s", providerErr.RetryAfter)
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
	} else if providerErr, ok := err.(*provider.Error); !ok || providerErr.Kind != provider.ErrorCredentials || !strings.Contains(providerErr.Error(), "401") {
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

func TestConnectorMapsForbiddenToCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := NewConnector(server.URL, "wrong").FetchChanges(context.Background(), connector.Credentials{}, time.Now().UTC(), nil)
	if providerErr, ok := err.(*provider.Error); !ok || providerErr.Kind != provider.ErrorCredentials {
		t.Fatalf("403 error = %#v, want credentials kind", err)
	}

	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer unavailable.Close()
	_, _, err = NewConnector(unavailable.URL, "secret").Fetch(context.Background(), connector.Credentials{}, nil)
	if providerErr, ok := err.(*provider.Error); !ok || providerErr.Kind != provider.ErrorUnavailable {
		t.Fatalf("502 error = %#v, want unavailable kind", err)
	}
}

func TestConnectorRefusesUnknownMajorVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"2.0","watermark":"2026-11-11T10:00:00Z","has_more":false,"books":[]}`))
	}))
	defer server.Close()

	_, _, err := NewConnector(server.URL, "secret").Fetch(context.Background(), connector.Credentials{}, nil)
	providerErr, ok := err.(*provider.Error)
	if !ok || providerErr.Kind != provider.ErrorInvalidSource || !strings.Contains(providerErr.Error(), "2.0") {
		t.Fatalf("version 2.0 error = %#v, want invalid_source naming the version", err)
	}
}

func TestConnectorRejectsHasMoreWithoutCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"1.0","watermark":"2026-11-11T10:00:00Z","has_more":true,"books":[]}`))
	}))
	defer server.Close()

	_, err := NewConnector(server.URL, "secret").FetchChanges(context.Background(), connector.Credentials{}, time.Now().UTC(), nil)
	providerErr, ok := err.(*provider.Error)
	if !ok || providerErr.Kind != provider.ErrorInvalidSource || !strings.Contains(providerErr.Error(), "cursor") {
		t.Fatalf("has_more without cursor error = %#v, want invalid_source", err)
	}
}

func TestConnectorRejectsOversizedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"version":"1.0","has_more":false,"books":[],"pad":"`))
		_, _ = w.Write([]byte(strings.Repeat("x", MaxResponseBytes)))
		_, _ = w.Write([]byte(`"}`))
	}))
	defer server.Close()

	_, _, err := NewConnector(server.URL, "secret").Fetch(context.Background(), connector.Credentials{}, nil)
	providerErr, ok := err.(*provider.Error)
	if !ok || providerErr.Kind != provider.ErrorInvalidSource || !strings.Contains(providerErr.Error(), "exceeds") {
		t.Fatalf("oversized body error = %#v, want invalid_source", err)
	}
}
