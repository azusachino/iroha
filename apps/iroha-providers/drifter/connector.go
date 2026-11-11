// Package drifter ingests the reading feed served by drifter-server, the
// owner's provider-neutral reading gateway (ADR 0011 amendment). The connector
// fetches authenticated, cursor-paginated pages; the adapter is a pure parser
// that turns a page into provider-neutral media observations.
package drifter

import (
	"context"
	"encoding/json"
	"errors"
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
	// SnapshotPath is the authenticated whole-shelf reading snapshot endpoint.
	SnapshotPath = "/api/v1/reading/snapshot"
	// ChangesPath is the authenticated incremental feed: books Drifter
	// recorded at or after a watermark.
	ChangesPath = "/api/v1/reading/changes"
	// DefaultLimit is the page size requested when the connector has none. It
	// stays small on purpose: upstream WeRead paces one request per second, so a
	// large page makes the gateway exceed its client timeout instead of
	// paginating.
	DefaultLimit = 10
	// ChangesLimit is the page size requested from the changes feed.
	ChangesLimit = 100
	// SourceInstanceKey names the one Drifter gateway a receipt came from.
	SourceInstanceKey = "drifter:primary"
	// IngestionModeSnapshot is the receipt mode recorded for a whole-shelf
	// page; every page of a snapshot run states the same mode.
	IngestionModeSnapshot = "full_snapshot"
	// IngestionModeChanges is the receipt mode recorded for a changes page.
	IngestionModeChanges = "incremental"
	// snapshotFilename names a stored snapshot page in evidence.
	snapshotFilename = "drifter_reading_snapshot.json"
	// changesFilename names a stored changes page in evidence.
	changesFilename = "drifter_reading_changes.json"
	// DefaultClientTimeout bounds one page fetch. A small page still pays
	// WeRead's per-request pacing on the gateway side, so the bound is
	// generous rather than tight.
	DefaultClientTimeout = 60 * time.Second
)

// ErrSnapshotRequired is the protocol signal that the requested watermark is
// outside the changes feed's horizon. It is the shared connector sentinel so a
// generic sync runner can recognize it without knowing this gateway.
var ErrSnapshotRequired = connector.ErrSnapshotRequired

var (
	_ connector.Connector     = Connector{}
	_ connector.ChangeFetcher = Connector{}
)

// Connector fetches one page of the Drifter reading feed per call. It does not
// parse or persist anything; the sync runner stores the body as raw evidence
// and the adapter parses it later.
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

// Fetch requests one page of the whole-shelf snapshot. A returned cursor means
// another page is available in the same run; nil means the snapshot is
// complete. The gateway names its account itself, so no account parameter is
// sent: the exported account stays legible in the page's own identity fields.
func (c Connector) Fetch(ctx context.Context, credentials connector.Credentials, cursor *connector.Cursor) (connector.Snapshot, *connector.Cursor, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(c.pageLimit()))
	if cursor != nil && cursor.Token != "" {
		query.Set("cursor", cursor.Token)
	}
	page, err := c.fetchPage(ctx, credentials, SnapshotPath, query, snapshotFilename, IngestionModeSnapshot)
	if err != nil {
		return connector.Snapshot{}, nil, err
	}
	return page.evidence, nextCursor(page), nil
}

// FetchChanges requests one page of the incremental feed: the books Drifter
// recorded at or after since. ErrSnapshotRequired means since is outside the
// feed's horizon and the caller must read a full snapshot instead.
func (c Connector) FetchChanges(ctx context.Context, credentials connector.Credentials, since time.Time, cursor *connector.Cursor) (connector.ChangePage, error) {
	if since.IsZero() {
		return connector.ChangePage{}, errors.New("drifter changes require a watermark")
	}
	query := url.Values{}
	query.Set("since", since.UTC().Format(time.RFC3339))
	query.Set("limit", strconv.Itoa(ChangesLimit))
	if cursor != nil && cursor.Token != "" {
		query.Set("cursor", cursor.Token)
	}
	page, err := c.fetchPage(ctx, credentials, ChangesPath, query, changesFilename, IngestionModeChanges)
	if err != nil {
		return connector.ChangePage{}, err
	}
	return connector.ChangePage{Snapshot: page.evidence, Watermark: page.Watermark, Cursor: nextCursor(page)}, nil
}

// fetchPage requests and decodes one feed page, attaching the exact response
// bytes as evidence. A 410, or a body that asks for a snapshot, means the
// requested watermark is no longer servable.
func (c Connector) fetchPage(ctx context.Context, credentials connector.Credentials, path string, query url.Values, filename, ingestionMode string) (ReadingSnapshot, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+path+"?"+query.Encode(), nil)
	if err != nil {
		return ReadingSnapshot{}, err
	}
	request.Header.Set("Accept", "application/json")
	if token := c.token(credentials); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: DefaultClientTimeout}
	}
	response, err := client.Do(request)
	if err != nil {
		return ReadingSnapshot{}, &provider.Error{Kind: provider.ErrorUnavailable, Provider: ProviderID, SourceKind: SourceKind, Op: "fetch", Err: err}
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return ReadingSnapshot{}, &provider.Error{Kind: provider.ErrorUnavailable, Provider: ProviderID, SourceKind: SourceKind, Op: "read_response", Err: err}
	}
	if response.StatusCode == http.StatusGone {
		return ReadingSnapshot{}, ErrSnapshotRequired
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return ReadingSnapshot{}, rateLimitError(response)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ReadingSnapshot{}, &provider.Error{Kind: provider.ErrorUnavailable, Provider: ProviderID, SourceKind: SourceKind, Op: "fetch", Err: fmt.Errorf("HTTP %s", response.Status)}
	}
	var page ReadingSnapshot
	if err := json.Unmarshal(body, &page); err != nil {
		return ReadingSnapshot{}, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: SourceKind, Op: "decode_response", Err: err}
	}
	if page.SnapshotRequired {
		return ReadingSnapshot{}, ErrSnapshotRequired
	}
	page.evidence = connector.Snapshot{
		ContentType:       "application/json",
		Body:              body,
		SourceKind:        SourceKind,
		Filename:          filename,
		SourceInstanceKey: SourceInstanceKey,
		IngestionMode:     ingestionMode,
		OrderingBasis:     page.Watermark,
		ObservedAt:        time.Now().UTC(),
	}
	return page, nil
}

func rateLimitError(response *http.Response) *provider.Error {
	rateLimitErr := &provider.Error{
		Kind:       provider.ErrorRateLimited,
		Provider:   ProviderID,
		SourceKind: SourceKind,
		Op:         "fetch",
		Err:        fmt.Errorf("HTTP 429 Retry-After=%s", response.Header.Get("Retry-After")),
	}
	if retryAfter, ok := provider.ParseRetryAfter(response.Header.Get("Retry-After"), time.Now().UTC()); ok {
		rateLimitErr.RetryAfter = &retryAfter
	}
	return rateLimitErr
}

func nextCursor(page ReadingSnapshot) *connector.Cursor {
	if !page.HasMore || page.Cursor == "" {
		return nil
	}
	return &connector.Cursor{Token: page.Cursor}
}

func (c Connector) pageLimit() int {
	if c.Limit > 0 {
		return c.Limit
	}
	return DefaultLimit
}

func (c Connector) baseURL() string {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return baseURL
}

func (c Connector) token(credentials connector.Credentials) string {
	if value := credentials.Values["token"]; value != "" {
		return value
	}
	return c.Token
}
