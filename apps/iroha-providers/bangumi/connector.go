package bangumi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

const (
	APIEndpoint      = "https://api.bgm.tv/v0/users"
	DefaultUserAgent = "iroha/0.1 (+https://github.com/azusachino/iroha)"
	DefaultLimit     = 50
)

type Connector struct {
	Username  string
	Token     string
	Endpoint  string
	UserAgent string
	Client    *http.Client
	Limit     int
}

func NewConnector(username, token string) Connector {
	return Connector{Username: username, Token: token, Endpoint: APIEndpoint, UserAgent: DefaultUserAgent, Client: &http.Client{Timeout: 30 * time.Second}, Limit: DefaultLimit}
}

func (c Connector) Descriptor() connector.Descriptor {
	return connector.Descriptor{ID: ProviderID, DisplayName: "Bangumi", SourceKind: SourceKind, RequiresAuth: c.Token != ""}
}

func (c Connector) Fetch(ctx context.Context, credentials connector.Credentials, cursor *connector.Cursor) (connector.Snapshot, *connector.Cursor, error) {
	username := c.Username
	if value := credentials.Values["username"]; value != "" {
		username = value
	}
	token := c.Token
	if value := credentials.Values["token"]; value != "" {
		token = value
	}
	if username == "" {
		return connector.Snapshot{}, nil, fmt.Errorf("bangumi username is required")
	}
	limit := c.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	page := 0
	if cursor != nil && cursor.Page > 0 {
		page = cursor.Page
	}
	offset := page * limit
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = APIEndpoint
	}
	requestURL := endpoint + "/" + url.PathEscape(username) + "/collections?limit=" + strconv.Itoa(limit) + "&offset=" + strconv.Itoa(offset)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return connector.Snapshot{}, nil, err
	}
	request.Header.Set("Accept", "application/json")
	userAgent := c.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	request.Header.Set("User-Agent", userAgent)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	client := c.Client
	if client == nil {
		client = http.DefaultClient
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
		providerErr := &provider.Error{Kind: provider.ErrorRateLimited, Provider: ProviderID, SourceKind: SourceKind, Op: "fetch", Err: fmt.Errorf("HTTP 429 Retry-After=%s", response.Header.Get("Retry-After"))}
		if hasRetryAfter {
			providerErr.RetryAfter = &retryAfter
		}
		return connector.Snapshot{}, nil, providerErr
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return connector.Snapshot{}, nil, &provider.Error{Kind: provider.ErrorUnavailable, Provider: ProviderID, SourceKind: SourceKind, Op: "fetch", Err: fmt.Errorf("HTTP %s", response.Status)}
	}
	var payload collectionsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return connector.Snapshot{}, nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: SourceKind, Op: "decode_response", Err: err}
	}
	if len(payload.Data) == 0 || offset+len(payload.Data) >= payload.Total {
		return connector.Snapshot{ContentType: "application/json", Body: body, SourceKind: SourceKind, Filename: "bangumi_page_" + strconv.Itoa(page) + ".json", ObservedAt: time.Now().UTC()}, nil, nil
	}
	next := &connector.Cursor{Page: page + 1}
	return connector.Snapshot{ContentType: "application/json", Body: body, SourceKind: SourceKind, Filename: "bangumi_page_" + strconv.Itoa(page) + ".json", ObservedAt: time.Now().UTC()}, next, nil
}
