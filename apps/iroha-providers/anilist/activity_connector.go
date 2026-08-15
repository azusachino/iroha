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
	ActivityConnectorID     = "anilist_activity"
	DefaultActivityLookback = 365 * 24 * time.Hour
	DefaultActivityPageSize = 50
	ActivityOverlap         = 24 * time.Hour
)

const (
	activityUserQuery  = `query($userName:String!){User(name:$userName){id}}`
	mediaActivityQuery = `query($userID:Int!,$page:Int!,$perPage:Int!,$createdAtGreater:Int!){
  Page(page:$page,perPage:$perPage){
    pageInfo{hasNextPage}
    activities(userId:$userID,createdAt_greater:$createdAtGreater,sort:[ID_DESC]){
      ... on ListActivity{id status progress createdAt media{id idMal type format episodes chapters volumes title{romaji english native} coverImage{large}}}
    }
  }
}`
)

type ActivityConnector struct {
	Username  string
	Token     string
	Endpoint  string
	UserAgent string
	Client    *http.Client
	PerPage   int
	Lookback  time.Duration
}

func NewActivityConnector(username, token string) ActivityConnector {
	return ActivityConnector{
		Username:  username,
		Token:     token,
		Endpoint:  APIEndpoint,
		UserAgent: DefaultUserAgent,
		Client:    &http.Client{Timeout: 30 * time.Second},
		PerPage:   DefaultActivityPageSize,
		Lookback:  DefaultActivityLookback,
	}
}

func (c ActivityConnector) Descriptor() connector.Descriptor {
	return connector.Descriptor{
		ID:           ActivityConnectorID,
		DisplayName:  "AniList activity",
		SourceKind:   ActivitySourceKind,
		RequiresAuth: c.Token != "",
	}
}

func (c ActivityConnector) Fetch(ctx context.Context, credentials connector.Credentials, cursor *connector.Cursor) (connector.Snapshot, *connector.Cursor, error) {
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
	createdAtGreater := time.Now().UTC().Add(-c.lookback()).Unix()
	userID := 0
	if cursor != nil {
		if cursor.Page > 0 {
			page = cursor.Page
		}
		if cursor.CreatedAfter > 0 {
			createdAtGreater = cursor.CreatedAfter
		}
		userID = cursor.UserID
	}
	if userID == 0 {
		resolvedUserID, resolveErr := c.resolveUserID(ctx, username, token)
		if resolveErr != nil {
			return connector.Snapshot{}, nil, resolveErr
		}
		userID = resolvedUserID
	}
	perPage := c.PerPage
	if perPage <= 0 {
		perPage = DefaultActivityPageSize
	}
	body, err := json.Marshal(map[string]any{
		"query": mediaActivityQuery,
		"variables": map[string]any{
			"userID":           userID,
			"page":             page,
			"perPage":          perPage,
			"createdAtGreater": createdAtGreater,
		},
	})
	if err != nil {
		return connector.Snapshot{}, nil, err
	}
	responseBody, err := c.post(ctx, token, body, "fetch")
	if err != nil {
		return connector.Snapshot{}, nil, err
	}
	var payload activityGraphQLResponse
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return connector.Snapshot{}, nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: ActivitySourceKind, Op: "decode_response", Err: err}
	}
	if len(payload.Errors) > 0 {
		return connector.Snapshot{}, nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: ActivitySourceKind, Op: "graphql", Err: fmt.Errorf("%s", payload.Errors[0].Message)}
	}
	next := (*connector.Cursor)(nil)
	if payload.Data.Page.PageInfo.HasNextPage && len(payload.Data.Page.Activities) > 0 {
		next = &connector.Cursor{Token: "activities", Page: page + 1, CreatedAfter: createdAtGreater, UserID: userID}
	}
	return connector.Snapshot{
		ContentType: "application/json",
		Body:        responseBody,
		SourceKind:  ActivitySourceKind,
		Filename:    "anilist_activity_page_" + strconv.Itoa(page) + ".json",
		ObservedAt:  time.Now().UTC(),
	}, next, nil
}

func (ActivityConnector) ResumeCursor() *connector.Cursor {
	return &connector.Cursor{
		Token:        "resume",
		CreatedAfter: time.Now().UTC().Add(-ActivityOverlap).Unix(),
	}
}

func (c ActivityConnector) resolveUserID(ctx context.Context, username, token string) (int, error) {
	body, err := json.Marshal(map[string]any{
		"query":     activityUserQuery,
		"variables": map[string]any{"userName": username},
	})
	if err != nil {
		return 0, err
	}
	responseBody, err := c.post(ctx, token, body, "resolve_user")
	if err != nil {
		return 0, err
	}
	var payload struct {
		Data struct {
			User *struct {
				ID int `json:"id"`
			} `json:"User"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return 0, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: ActivitySourceKind, Op: "decode_user", Err: err}
	}
	if len(payload.Errors) > 0 {
		return 0, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: ActivitySourceKind, Op: "graphql_user", Err: fmt.Errorf("%s", payload.Errors[0].Message)}
	}
	if payload.Data.User == nil || payload.Data.User.ID == 0 {
		return 0, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: ActivitySourceKind, Op: "resolve_user", Err: fmt.Errorf("AniList user %q was not found", username)}
	}
	return payload.Data.User.ID, nil
}

func (c ActivityConnector) post(ctx context.Context, token string, body []byte, op string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint(c.Endpoint), bytes.NewReader(body))
	if err != nil {
		return nil, err
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
		return nil, &provider.Error{Kind: provider.ErrorUnavailable, Provider: ProviderID, SourceKind: ActivitySourceKind, Op: op, Err: err}
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, &provider.Error{Kind: provider.ErrorUnavailable, Provider: ProviderID, SourceKind: ActivitySourceKind, Op: "read_" + op, Err: err}
	}
	if response.StatusCode == http.StatusTooManyRequests {
		retryAfter, hasRetryAfter := provider.ParseRetryAfter(response.Header.Get("Retry-After"), time.Now().UTC())
		providerErr := &provider.Error{Kind: provider.ErrorRateLimited, Provider: ProviderID, SourceKind: ActivitySourceKind, Op: op, Err: fmt.Errorf("HTTP 429 Retry-After=%s", response.Header.Get("Retry-After"))}
		if hasRetryAfter {
			providerErr.RetryAfter = &retryAfter
		}
		return nil, providerErr
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, &provider.Error{Kind: provider.ErrorUnavailable, Provider: ProviderID, SourceKind: ActivitySourceKind, Op: op, Err: fmt.Errorf("HTTP %s", response.Status)}
	}
	return responseBody, nil
}

func (c ActivityConnector) lookback() time.Duration {
	if c.Lookback <= 0 {
		return DefaultActivityLookback
	}
	return c.Lookback
}
