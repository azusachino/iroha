package anilist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

const connectorFixture = `{"data":{"MediaListCollection":{"lists":[{"entries":[{"id":1,"status":"CURRENT","media":{"id":20,"type":"ANIME","format":"TV","title":{"native":"NARUTO"}}}]}]}}}`

func TestConnectorFetchSendsAuthAndAdvancesAnimeToManga(t *testing.T) {
	var gotUserAgent, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(connectorFixture))
	}))
	defer server.Close()

	client := NewConnector("azusachino", "secret")
	client.Endpoint = server.URL
	client.PerChunk = 50
	snapshot, next, err := client.Fetch(context.Background(), connector.Credentials{}, nil)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if snapshot.SourceKind != SourceKind || snapshot.ContentType != "application/json" || len(snapshot.Body) == 0 {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if next == nil || next.Token != "MANGA" || next.Page != 1 {
		t.Fatalf("next cursor = %#v, want manga page 1", next)
	}
	if gotUserAgent == "" || gotAuth != "Bearer secret" {
		t.Fatalf("headers user-agent=%q auth=%q", gotUserAgent, gotAuth)
	}
}

func TestConnectorMapsRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "12")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewConnector("azusachino", "")
	client.Endpoint = server.URL
	_, _, err := client.Fetch(context.Background(), connector.Credentials{}, nil)
	if err == nil {
		t.Fatal("Fetch() returned nil error for 429")
	}
	providerErr, ok := err.(*provider.Error)
	if !ok || providerErr.Kind != provider.ErrorRateLimited || !strings.Contains(providerErr.Error(), "Retry-After=12") {
		t.Fatalf("rate limit error = %#v", err)
	}
	if providerErr.RetryAfter == nil || *providerErr.RetryAfter != 12*time.Second {
		t.Fatalf("retry after = %v, want 12s", providerErr.RetryAfter)
	}
}

func TestLivePublicUsername(t *testing.T) {
	if os.Getenv("IROHA_ANILIST_LIVE") != "1" {
		t.Skip("set IROHA_ANILIST_LIVE=1 to run the public AniList smoke")
	}
	username := os.Getenv("IROHA_ANILIST_USERNAME")
	if username == "" {
		t.Fatal("IROHA_ANILIST_USERNAME is required for the live smoke")
	}
	snapshot, _, err := NewConnector(username, "").Fetch(context.Background(), connector.Credentials{}, nil)
	if err != nil {
		t.Fatalf("live AniList fetch: %v", err)
	}
	entries, err := ParseSnapshot(snapshot.Body)
	if err != nil {
		t.Fatalf("decode live AniList snapshot: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("live AniList snapshot returned no entries")
	}
}
