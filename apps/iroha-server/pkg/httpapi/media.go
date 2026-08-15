package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mediaListResponse struct {
	Items        []mediaResponse `json:"items"`
	NextCursor   *string         `json:"next_cursor"`
	HasMore      bool            `json:"has_more"`
	StatusCounts map[string]int  `json:"status_counts"`
	ActiveCount  int             `json:"active_count"`
}

type mediaResponse struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	MediaType          string    `json:"media_type"`
	ItemRole           string    `json:"item_role"`
	CoverImageURL      string    `json:"cover_image_url,omitempty"`
	Status             *string   `json:"status,omitempty"`
	Position           *float64  `json:"position,omitempty"`
	Total              *float64  `json:"total,omitempty"`
	Unit               *string   `json:"unit,omitempty"`
	ProgressPercent    *float64  `json:"progress_percent,omitempty"`
	LastUpdateAt       time.Time `json:"last_update_at"`
	Rating             *float64  `json:"rating,omitempty"`
	HiddenFromContinue bool      `json:"hidden_from_continue"`
	NativeTitle        *string   `json:"native_title,omitempty"`
	EpisodeCount       *int      `json:"episode_count,omitempty"`
	ChapterCount       *int      `json:"chapter_count,omitempty"`
	StartedOn          *string   `json:"started_on,omitempty"`
	CompletedOn        *string   `json:"completed_on,omitempty"`
}

type mediaDetailResponse struct {
	Item      mediaResponse           `json:"item"`
	Work      mediaWorkResponse       `json:"work"`
	Progress  *mediaProgressResponse  `json:"progress,omitempty"`
	Creators  []mediaCreatorResponse  `json:"creators"`
	Relations []mediaRelationResponse `json:"relations"`
	Events    []mediaEventResponse    `json:"events"`
	Updates   []mediaChangeResponse   `json:"updates"`
}

type mediaWorkResponse struct {
	ID               string     `json:"id"`
	WorkKind         string     `json:"work_kind"`
	PrimaryTitle     string     `json:"primary_title"`
	OriginalTitle    string     `json:"original_title"`
	OriginalLanguage string     `json:"original_language"`
	FirstReleaseDate *time.Time `json:"first_release_date,omitempty"`
	Description      string     `json:"description"`
}

type mediaProgressResponse struct {
	Status          string     `json:"status"`
	Unit            string     `json:"unit"`
	Position        *float64   `json:"position,omitempty"`
	Total           *float64   `json:"total,omitempty"`
	ProgressPercent *float64   `json:"progress_percent,omitempty"`
	StartedOn       *string    `json:"started_on,omitempty"`
	LastUpdateAt    *time.Time `json:"last_update_at,omitempty"`
	CompletedOn     *string    `json:"completed_on,omitempty"`
	PlayCount       int        `json:"play_count"`
}

type mediaCreatorResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type mediaRelationResponse struct {
	ID            string `json:"id"`
	RelationType  string `json:"relation_type"`
	Direction     string `json:"direction"`
	RelatedItemID string `json:"related_item_id"`
	RelatedTitle  string `json:"related_title"`
	RelatedType   string `json:"related_type"`
	CoverImageURL string `json:"cover_image_url,omitempty"`
}

type mediaEventResponse struct {
	ID              string    `json:"id"`
	EventType       string    `json:"event_type"`
	EventAt         time.Time `json:"event_at"`
	Unit            string    `json:"unit,omitempty"`
	Position        *float64  `json:"position,omitempty"`
	Total           *float64  `json:"total,omitempty"`
	ProgressPercent *float64  `json:"progress_percent,omitempty"`
	Rating          *float64  `json:"rating,omitempty"`
	Note            string    `json:"note,omitempty"`
}

type mediaCreateEventRequest struct {
	MediaID         string    `json:"media_id"`
	EventType       string    `json:"event_type"`
	EventAt         time.Time `json:"event_at"`
	SourceKind      string    `json:"source_kind"`
	IdempotencyKey  string    `json:"idempotency_key"`
	Unit            string    `json:"unit"`
	Position        *float64  `json:"position"`
	Total           *float64  `json:"total"`
	ProgressPercent *float64  `json:"progress_percent"`
	Rating          *float64  `json:"rating"`
	RatingScale     *float64  `json:"rating_scale"`
	Note            string    `json:"note"`
}

type mediaChangeListResponse struct {
	Items      []mediaChangeResponse `json:"items"`
	NextCursor *string               `json:"next_cursor"`
	HasMore    bool                  `json:"has_more"`
}

type mediaChangeResponse struct {
	ID                 string     `json:"id"`
	MediaID            string     `json:"media_id"`
	Title              string     `json:"title"`
	NativeTitle        *string    `json:"native_title,omitempty"`
	CoverImageURL      string     `json:"cover_image_url,omitempty"`
	SourceKind         string     `json:"source_kind"`
	ChangeKind         string     `json:"change_kind"`
	TimeBasis          string     `json:"time_basis"`
	ObservedAt         time.Time  `json:"observed_at"`
	EffectiveAt        *time.Time `json:"effective_at,omitempty"`
	EffectiveOn        *string    `json:"effective_on,omitempty"`
	DatePrecision      string     `json:"date_precision,omitempty"`
	ProviderRecordedAt *time.Time `json:"provider_recorded_at,omitempty"`
	Status             string     `json:"status,omitempty"`
	Unit               string     `json:"unit,omitempty"`
	Position           *float64   `json:"position,omitempty"`
	Total              *float64   `json:"total,omitempty"`
	ProgressPercent    *float64   `json:"progress_percent,omitempty"`
	Rating             *float64   `json:"rating,omitempty"`
	Note               string     `json:"note,omitempty"`
	RepeatCount        int        `json:"repeat_count"`
}

type mediaEventListResponse struct {
	Items      []mediaHomeEventResponse `json:"items"`
	NextCursor *string                  `json:"next_cursor"`
	HasMore    bool                     `json:"has_more"`
}

type mediaHomeEventResponse struct {
	ID              string    `json:"id"`
	MediaID         string    `json:"media_id"`
	Title           string    `json:"title"`
	NativeTitle     *string   `json:"native_title,omitempty"`
	CoverImageURL   string    `json:"cover_image_url,omitempty"`
	EventType       string    `json:"event_type"`
	OccurredAt      time.Time `json:"occurred_at"`
	Unit            string    `json:"unit,omitempty"`
	Position        *float64  `json:"position,omitempty"`
	Total           *float64  `json:"total,omitempty"`
	ProgressPercent *float64  `json:"progress_percent,omitempty"`
	Rating          *float64  `json:"rating,omitempty"`
}

func (s *Server) handleListMedia(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseMediaFilters(w, r)
	if !ok {
		return
	}
	page, err := s.deps.MediaService.List(filters)
	if err != nil {
		s.deps.Logger.Error("list media", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list media")
		return
	}

	items := make([]mediaResponse, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, toMediaResponse(row))
	}
	response := mediaListResponse{
		Items: items, HasMore: page.HasMore,
		StatusCounts: page.StatusCounts, ActiveCount: page.ActiveCount,
	}
	if page.NextCursor != nil {
		cursor := media.EncodeCursor(*page.NextCursor)
		response.NextCursor = &cursor
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleMediaAggregates(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseMediaFilters(w, r)
	if !ok {
		return
	}
	aggregates, err := s.deps.MediaService.AggregatesFiltered(time.Now().UTC(), filters)
	if err != nil {
		s.deps.Logger.Error("aggregate media", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to aggregate media")
		return
	}
	writeJSON(w, http.StatusOK, aggregates)
}

func (s *Server) handleListMediaEvents(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseMediaEventFilters(w, r)
	if !ok {
		return
	}
	page, err := s.deps.MediaService.Events(filters)
	if err != nil {
		s.deps.Logger.Error("list media events", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list media events")
		return
	}
	items := make([]mediaHomeEventResponse, 0, len(page.Items))
	for _, event := range page.Items {
		items = append(items, mediaHomeEventResponse{
			ID: mediaEventID(event.ID), MediaID: ids.Encode(ids.MediaPrefix, event.MediaItemID),
			Title: event.Title, NativeTitle: event.NativeTitle, CoverImageURL: event.CoverImageURL, EventType: event.EventType,
			OccurredAt: event.OccurredAt, Unit: event.Unit, Position: event.Position, Total: event.Total,
			ProgressPercent: event.ProgressPercent, Rating: normalizedRating(event.Rating, event.RatingScale),
		})
	}
	response := mediaEventListResponse{Items: items, HasMore: page.HasMore}
	if page.NextCursor != nil {
		cursor := media.EncodeCursor(*page.NextCursor)
		response.NextCursor = &cursor
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateMediaEvent(w http.ResponseWriter, r *http.Request) {
	if s.deps.MediaService == nil {
		writeError(w, http.StatusServiceUnavailable, "media service unavailable")
		return
	}
	var request mediaCreateEventRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	mediaID, err := ids.Decode(ids.MediaPrefix, request.MediaID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid media id")
		return
	}
	event, err := s.deps.MediaService.CreateEvent(media.CreateEventInput{
		MediaItemID: mediaID, EventType: request.EventType, EventAt: request.EventAt,
		SourceKind: request.SourceKind, SourceEventID: request.IdempotencyKey,
		Unit: request.Unit, Position: request.Position, Total: request.Total,
		ProgressPercent: request.ProgressPercent, Rating: request.Rating,
		RatingScale: request.RatingScale, Note: request.Note,
	})
	if err != nil {
		switch err {
		case media.ErrMediaItemNotFound:
			writeError(w, http.StatusNotFound, err.Error())
		case media.ErrEventConflict:
			writeError(w, http.StatusConflict, err.Error())
		default:
			if errors.Is(err, media.ErrEventAtRequired) || errors.Is(err, media.ErrInvalidEventType) || errors.Is(err, media.ErrSourceEventIDRequired) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			s.deps.Logger.Error("create media event", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to create media event")
		}
		return
	}
	if s.deps.Cache != nil {
		if err := s.deps.Cache.InvalidateChange(r.Context(), cache.ChangeMedia); err != nil {
			s.deps.Logger.Error("invalidate caches after media event", "error", err)
		}
	}
	writeJSON(w, http.StatusOK, mediaEventResponse{
		ID: mediaEventID(event.ID), EventType: event.EventType, EventAt: event.OccurredAt,
		Unit: event.Unit, Position: event.Position, Total: event.Total,
		ProgressPercent: event.ProgressPercent, Rating: normalizedRating(event.Rating, event.RatingScale),
	})
}

func (s *Server) handleListMediaChanges(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseMediaEventFilters(w, r)
	if !ok {
		return
	}
	page, err := s.deps.MediaService.Changes(media.ChangeListFilters{
		From: filters.From, To: filters.To, Limit: filters.Limit, Cursor: filters.Cursor,
	})
	if err != nil {
		s.deps.Logger.Error("list media changes", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list media changes")
		return
	}
	response := mediaChangeListResponse{Items: make([]mediaChangeResponse, 0, len(page.Items)), HasMore: page.HasMore}
	for _, change := range page.Items {
		response.Items = append(response.Items, toMediaChangeResponse(change))
	}
	if page.NextCursor != nil {
		cursor := media.EncodeCursor(*page.NextCursor)
		response.NextCursor = &cursor
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetMedia(w http.ResponseWriter, r *http.Request) {
	id, err := ids.Decode(ids.MediaPrefix, chi.URLParam(r, "mediaId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid media id")
		return
	}
	detail, found, err := s.deps.MediaService.Get(id)
	if err != nil {
		s.deps.Logger.Error("get media", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get media")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "media not found")
		return
	}
	relations := make([]mediaRelationResponse, 0, len(detail.Relations))
	for _, relation := range detail.Relations {
		relations = append(relations, mediaRelationResponse{
			ID: ids.Encode(ids.MediaPrefix, relation.ID), RelationType: relation.RelationType,
			Direction: relation.Direction, RelatedItemID: ids.Encode(ids.MediaPrefix, relation.RelatedItemID),
			RelatedTitle: relation.RelatedTitle, RelatedType: relation.RelatedType,
			CoverImageURL: relation.CoverImageURL,
		})
	}
	events := make([]mediaEventResponse, 0, len(detail.Events))
	for _, event := range detail.Events {
		events = append(events, mediaEventResponse{
			ID: mediaEventID(event.ID), EventType: event.EventType, EventAt: event.EventAt,
			Unit: event.Unit, Position: event.Position, Total: event.Total,
			ProgressPercent: event.ProgressPercent, Rating: normalizedRating(event.Rating, event.RatingScale),
			Note: event.Note,
		})
	}
	creators := make([]mediaCreatorResponse, 0, len(detail.Creators))
	for _, creator := range detail.Creators {
		creators = append(creators, mediaCreatorResponse{
			ID: ids.Encode(ids.MediaPrefix, creator.ID), Name: creator.Name, Role: creator.Role,
		})
	}
	updates := make([]mediaChangeResponse, 0, len(detail.Updates))
	for _, change := range detail.Updates {
		updates = append(updates, toMediaChangeResponse(change))
	}
	var progress *mediaProgressResponse
	if detail.Progress != nil {
		progress = &mediaProgressResponse{
			Status: detail.Progress.Status, Unit: detail.Progress.Unit,
			Position: detail.Progress.Position, Total: detail.Progress.Total,
			ProgressPercent: detail.Progress.ProgressPercent,
			StartedOn:       partialDateString(detail.Progress.StartedOnValue, detail.Progress.StartedOnPrecision),
			LastUpdateAt:    detail.Progress.LastUpdateAt,
			CompletedOn:     partialDateString(detail.Progress.CompletedOnValue, detail.Progress.CompletedOnPrecision),
			PlayCount:       detail.Progress.PlayCount,
		}
	}
	writeJSON(w, http.StatusOK, mediaDetailResponse{
		Item: toMediaResponse(detail.Item),
		Work: mediaWorkResponse{
			ID: ids.Encode(ids.MediaPrefix, detail.Work.ID), WorkKind: detail.Work.WorkKind,
			PrimaryTitle: detail.Work.PrimaryTitle, OriginalTitle: detail.Work.OriginalTitle,
			OriginalLanguage: detail.Work.OriginalLanguage, FirstReleaseDate: detail.Work.FirstReleaseDate,
			Description: detail.Work.Description,
		},
		Progress: progress, Creators: creators, Relations: relations, Events: events, Updates: updates,
	})
}

func mediaEventID(id uuid.UUID) string {
	return ids.Encode(ids.MediaEventPrefix, id)
}

func parseMediaFilters(w http.ResponseWriter, r *http.Request) (media.ListFilters, bool) {
	query := r.URL.Query()
	limit, ok := parsePageLimit(w, r)
	if !ok {
		return media.ListFilters{}, false
	}
	filters := media.ListFilters{
		Status:    query.Get("status"),
		MediaType: query.Get("media_type"),
		Family:    query.Get("family"),
		Limit:     limit,
	}
	if filters.Family != "" && !media.IsFamily(filters.Family) {
		writeError(w, http.StatusBadRequest, "invalid family")
		return media.ListFilters{}, false
	}
	if value := query.Get("completed_year"); value != "" {
		year, err := strconv.Atoi(value)
		if err != nil || year < 1 || year > 9999 {
			writeError(w, http.StatusBadRequest, "invalid completed_year")
			return media.ListFilters{}, false
		}
		filters.CompletedYear = &year
	}
	if value := query.Get("cursor"); value != "" {
		cursor, err := media.DecodeCursor(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return media.ListFilters{}, false
		}
		filters.Cursor = &cursor
	}
	return filters, true
}

func parseMediaEventFilters(w http.ResponseWriter, r *http.Request) (media.EventListFilters, bool) {
	query := r.URL.Query()
	limit, ok := parsePageLimit(w, r)
	if !ok {
		return media.EventListFilters{}, false
	}
	filters := media.EventListFilters{Limit: limit}
	for key, destination := range map[string]**time.Time{"from": &filters.From, "to": &filters.To} {
		if value := query.Get(key); value != "" {
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid "+key)
				return media.EventListFilters{}, false
			}
			if key == "to" {
				parsed = parsed.Add(24 * time.Hour)
			}
			*destination = &parsed
		}
	}
	if value := query.Get("cursor"); value != "" {
		cursor, err := media.DecodeCursor(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return media.EventListFilters{}, false
		}
		filters.Cursor = &cursor
	}
	return filters, true
}

func toMediaResponse(row media.Item) mediaResponse {
	return mediaResponse{
		ID:                 ids.Encode(ids.MediaPrefix, row.ID),
		Title:              row.Title,
		MediaType:          row.MediaType,
		ItemRole:           row.ItemRole,
		CoverImageURL:      row.CoverImageURL,
		Status:             row.Status,
		Position:           row.Position,
		Total:              row.Total,
		Unit:               row.Unit,
		ProgressPercent:    row.ProgressPercent,
		LastUpdateAt:       row.LastUpdateAt,
		Rating:             normalizedRating(row.Rating, row.RatingScale),
		HiddenFromContinue: row.HiddenFromContinue,
		NativeTitle:        row.NativeTitle,
		EpisodeCount:       row.EpisodeCount,
		ChapterCount:       row.ChapterCount,
		StartedOn:          partialDateString(row.StartedOnValue, row.StartedOnPrecision),
		CompletedOn:        partialDateString(row.CompletedOnValue, row.CompletedOnPrecision),
	}
}

func toMediaChangeResponse(change media.Change) mediaChangeResponse {
	return mediaChangeResponse{
		ID: ids.Encode(ids.MediaChangePrefix, change.ID), MediaID: ids.Encode(ids.MediaPrefix, change.MediaItemID),
		Title: change.Title, NativeTitle: change.NativeTitle, CoverImageURL: change.CoverImageURL,
		SourceKind: change.SourceKind, ChangeKind: change.ChangeKind, TimeBasis: change.TimeBasis,
		ObservedAt: change.ObservedAt, EffectiveAt: change.EffectiveAt,
		EffectiveOn:   partialDateString(change.EffectiveOnValue, change.EffectiveOnPrecision),
		DatePrecision: change.EffectiveOnPrecision, ProviderRecordedAt: change.ProviderRecordedAt,
		Status: change.Status, Unit: change.Unit, Position: change.Position, Total: change.Total,
		ProgressPercent: change.ProgressPercent, Rating: normalizedRating(change.Rating, change.RatingScale),
		Note: change.Note, RepeatCount: change.RepeatCount,
	}
}

func partialDateString(value *time.Time, precision string) *string {
	if value == nil {
		return nil
	}
	var format string
	switch precision {
	case "year":
		format = "2006"
	case "month":
		format = "2006-01"
	case "day":
		format = "2006-01-02"
	default:
		return nil
	}
	result := value.UTC().Format(format)
	return &result
}

func normalizedRating(rating, scale *float64) *float64 {
	if rating == nil || scale == nil || *scale <= 0 {
		return nil
	}
	normalized := *rating / *scale * 10
	return &normalized
}
