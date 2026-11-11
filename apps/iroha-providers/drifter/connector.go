// Package drifter ingests the reading snapshots served by drifter-server, the
// owner's provider-neutral reading gateway (ADR 0011 amendment). The connector
// fetches authenticated, cursor-paginated pages; the adapter is a pure parser
// that turns a page into provider-neutral media observations.
package drifter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

const (
	ProviderID = "drifter"
	SourceKind = coreimports.KindDrifter

	// DefaultBaseURL is drifter-server's local listen address.
	DefaultBaseURL = "http://127.0.0.1:8765"
	// SnapshotPath is the authenticated reading snapshot endpoint.
	SnapshotPath = "/api/v1/reading/snapshot"
	// DefaultLimit is the page size requested when the connector has none.
	DefaultLimit = 50
	// DefaultAccount is the single source account the gateway exports for the
	// owner. It is explicit in the request so the exported account is legible
	// in evidence, not implied by a server default.
	DefaultAccount = "default"
	// SourceInstanceKey names the one Drifter gateway a receipt came from.
	SourceInstanceKey = "drifter:primary"
	// IngestionMode is the receipt mode recorded per snapshot page. A sync run
	// is a full shelf snapshot; the run-level coverage assertion states that,
	// while each page is stored as its own bounded raw file.
	IngestionMode = "full_snapshot"
	// DefaultClientTimeout bounds one page fetch.
	DefaultClientTimeout = 30 * time.Second
)

// Connector fetches one page of the Drifter reading snapshot per call. It does
// not parse or persist anything; the sync runner stores the body as raw
// evidence and the adapter parses it later.
type Connector struct {
	BaseURL string
	Token   string
	Client  *http.Client
	Limit   int
}

// NewConnector builds a connector for one Drifter gateway. An empty baseURL
// falls back to DefaultBaseURL so a local gateway needs no configuration.
func NewConnector(baseURL, token string) Connector {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return Connector{BaseURL: baseURL, Token: token, Client: &http.Client{Timeout: DefaultClientTimeout}, Limit: DefaultLimit}
}

func (c Connector) Descriptor() connector.Descriptor {
	return connector.Descriptor{ID: ProviderID, DisplayName: "Drifter", SourceKind: SourceKind, RequiresAuth: true}
}

// Fetch requests one snapshot page. A returned cursor means another page is
// available in the same run; nil means the snapshot is complete.
func (c Connector) Fetch(ctx context.Context, credentials connector.Credentials, cursor *connector.Cursor) (connector.Snapshot, *connector.Cursor, error) {
	limit := c.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	token := c.Token
	if value := credentials.Values["token"]; value != "" {
		token = value
	}

	requestURL := baseURL + SnapshotPath + "?account=" + url.QueryEscape(DefaultAccount) + "&limit=" + strconv.Itoa(limit)
	if cursor != nil && cursor.Token != "" {
		requestURL += "&cursor=" + url.QueryEscape(cursor.Token)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return connector.Snapshot{}, nil, err
	}
	request.Header.Set("Accept", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: DefaultClientTimeout}
	}
	response, err := client.Do(request)
	if err != nil {
		return connector.Snapshot{}, nil, &provider.Error{Kind: provider.ErrorUnavailable, Provider: ProviderID, SourceKind: SourceKind, Op: "fetch", Err: err}
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return connector.Snapshot{}, nil, &provider.Error{Kind: provider.ErrorUnavailable, Provider: ProviderID, SourceKind: SourceKind, Op: "read_response", Err: err}
	}
	if response.StatusCode == http.StatusTooManyRequests {
		retryAfter, hasRetryAfter := provider.ParseRetryAfter(response.Header.Get("Retry-After"), time.Now().UTC())
		rateLimitErr := &provider.Error{Kind: provider.ErrorRateLimited, Provider: ProviderID, SourceKind: SourceKind, Op: "fetch", Err: fmt.Errorf("HTTP 429 Retry-After=%s", response.Header.Get("Retry-After"))}
		if hasRetryAfter {
			rateLimitErr.RetryAfter = &retryAfter
		}
		return connector.Snapshot{}, nil, rateLimitErr
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return connector.Snapshot{}, nil, &provider.Error{Kind: provider.ErrorUnavailable, Provider: ProviderID, SourceKind: SourceKind, Op: "fetch", Err: fmt.Errorf("HTTP %s", response.Status)}
	}
	var envelope readingSnapshotResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return connector.Snapshot{}, nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: SourceKind, Op: "decode_response", Err: err}
	}
	snapshot := connector.Snapshot{
		ContentType:       "application/json",
		Body:              body,
		SourceKind:        SourceKind,
		Filename:          "drifter_reading_snapshot.json",
		SourceInstanceKey: SourceInstanceKey,
		IngestionMode:     IngestionMode,
		ObservedAt:        time.Now().UTC(),
	}
	if envelope.HasMore && envelope.Cursor != "" {
		return snapshot, &connector.Cursor{Token: envelope.Cursor}, nil
	}
	return snapshot, nil, nil
}
