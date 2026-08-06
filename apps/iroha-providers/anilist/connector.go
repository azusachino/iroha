package anilist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

const (
	APIEndpoint      = "https://graphql.anilist.co"
	DefaultUserAgent = "iroha/0.1 (+https://github.com/azusachino/iroha)"
	DefaultPerChunk  = 50
)

const mediaListQuery = `query($userName:String!,$type:MediaType!,$chunk:Int!,$perChunk:Int!){
  User(name:$userName){mediaListOptions{scoreFormat}}
  MediaListCollection(userName:$userName,type:$type,chunk:$chunk,perChunk:$perChunk){
    lists{entries{id status score progress progressVolumes repeat notes updatedAt startedAt{year month day} completedAt{year month day}
      media{id idMal type format episodes chapters volumes startDate{year month day} title{romaji english native} coverImage{large}
        relations{edges{relationType node{id}}}}}}
  }
}`

type Connector struct {
	Username  string
	Token     string
	Endpoint  string
	UserAgent string
	Client    *http.Client
	PerChunk  int
}

func NewConnector(username, token string) Connector {
	return Connector{Username: username, Token: token, Endpoint: APIEndpoint, UserAgent: DefaultUserAgent, Client: &http.Client{Timeout: 30 * time.Second}, PerChunk: DefaultPerChunk}
}

func (c Connector) Descriptor() connector.Descriptor {
	return connector.Descriptor{ID: ProviderID, DisplayName: "AniList", SourceKind: SourceKind, RequiresAuth: c.Token != ""}
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
		return connector.Snapshot{}, nil, fmt.Errorf("anilist username is required")
	}
	page := 1
	mediaType := "ANIME"
	if cursor != nil {
		if cursor.Page > 0 {
			page = cursor.Page
		}
		if cursor.Token != "" {
			mediaType = cursor.Token
		}
	}
	perChunk := c.PerChunk
	if perChunk <= 0 {
		perChunk = DefaultPerChunk
	}
	body, err := json.Marshal(map[string]any{
		"query":     mediaListQuery,
		"variables": map[string]any{"userName": username, "type": mediaType, "chunk": page, "perChunk": perChunk},
	})
	if err != nil {
		return connector.Snapshot{}, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint(c.Endpoint), bytes.NewReader(body))
	if err != nil {
		return connector.Snapshot{}, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
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
	responseBody, err := io.ReadAll(response.Body)
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
	var payload graphQLResponse
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return connector.Snapshot{}, nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: SourceKind, Op: "decode_response", Err: err}
	}
	if len(payload.Errors) > 0 {
		return connector.Snapshot{}, nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: SourceKind, Op: "graphql", Err: fmt.Errorf("%s", payload.Errors[0].Message)}
	}
	count := countEntries(payload)
	next := nextCursor(mediaType, page, count, perChunk)
	return connector.Snapshot{ContentType: "application/json", Body: responseBody, SourceKind: SourceKind, Filename: "anilist_" + mediaType + "_chunk_" + strconv.Itoa(page) + ".json"}, next, nil
}

func endpoint(value string) string {
	if value == "" {
		return APIEndpoint
	}
	return value
}

func countEntries(payload graphQLResponse) int {
	if payload.Data.MediaListCollection == nil {
		return 0
	}
	count := 0
	for _, list := range payload.Data.MediaListCollection.Lists {
		count += len(list.Entries)
	}
	return count
}

func nextCursor(mediaType string, page, count, perChunk int) *connector.Cursor {
	if count >= perChunk {
		return &connector.Cursor{Token: mediaType, Page: page + 1}
	}
	if mediaType == "ANIME" {
		return &connector.Cursor{Token: "MANGA", Page: 1}
	}
	return nil
}
