package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/go-chi/chi/v5"
)

type mediaListResponse struct {
	Items      []mediaResponse `json:"items"`
	NextCursor *string         `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}

type mediaResponse struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	MediaType       string    `json:"media_type"`
	ItemRole        string    `json:"item_role"`
	CoverImageURL   string    `json:"cover_image_url,omitempty"`
	Status          *string   `json:"status,omitempty"`
	Position        *float64  `json:"position,omitempty"`
	Total           *float64  `json:"total,omitempty"`
	ProgressPercent *float64  `json:"progress_percent,omitempty"`
	LastUpdateAt    time.Time `json:"last_update_at"`
	Rating          *float64  `json:"rating,omitempty"`
}

type mediaDetailResponse struct {
	Item      mediaResponse           `json:"item"`
	Work      mediaWorkResponse       `json:"work"`
	Progress  *mediaProgressResponse  `json:"progress,omitempty"`
	Creators  []mediaCreatorResponse  `json:"creators"`
	Relations []mediaRelationResponse `json:"relations"`
	Events    []mediaEventResponse    `json:"events"`
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
	StartedAt       *time.Time `json:"started_at,omitempty"`
	LastUpdateAt    *time.Time `json:"last_update_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
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
	ID              string     `json:"id"`
	EventType       string     `json:"event_type"`
	EventAt         *time.Time `json:"event_at,omitempty"`
	Unit            string     `json:"unit,omitempty"`
	Position        *float64   `json:"position,omitempty"`
	Total           *float64   `json:"total,omitempty"`
	ProgressPercent *float64   `json:"progress_percent,omitempty"`
	Rating          *float64   `json:"rating,omitempty"`
	Note            string     `json:"note,omitempty"`
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
	response := mediaListResponse{Items: items, HasMore: page.HasMore}
	if page.NextCursor != nil {
		cursor := media.EncodeCursor(*page.NextCursor)
		response.NextCursor = &cursor
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleMediaAggregates(w http.ResponseWriter, _ *http.Request) {
	aggregates, err := s.deps.MediaService.Aggregates(time.Now().UTC())
	if err != nil {
		s.deps.Logger.Error("aggregate media", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to aggregate media")
		return
	}
	writeJSON(w, http.StatusOK, aggregates)
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
			ID: ids.Encode(ids.MediaPrefix, event.ID), EventType: event.EventType, EventAt: event.EventAt,
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
	var progress *mediaProgressResponse
	if detail.Progress != nil {
		progress = &mediaProgressResponse{
			Status: detail.Progress.Status, Unit: detail.Progress.Unit,
			Position: detail.Progress.Position, Total: detail.Progress.Total,
			ProgressPercent: detail.Progress.ProgressPercent, StartedAt: detail.Progress.StartedAt,
			LastUpdateAt: detail.Progress.LastUpdateAt, FinishedAt: detail.Progress.FinishedAt,
			PlayCount: detail.Progress.PlayCount,
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
		Progress: progress, Creators: creators, Relations: relations, Events: events,
	})
}

func parseMediaFilters(w http.ResponseWriter, r *http.Request) (media.ListFilters, bool) {
	query := r.URL.Query()
	filters := media.ListFilters{
		Status:    query.Get("status"),
		MediaType: query.Get("media_type"),
	}
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return media.ListFilters{}, false
		}
		filters.Limit = limit
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

func toMediaResponse(row media.Item) mediaResponse {
	return mediaResponse{
		ID:              ids.Encode(ids.MediaPrefix, row.ID),
		Title:           row.Title,
		MediaType:       row.MediaType,
		ItemRole:        row.ItemRole,
		CoverImageURL:   row.CoverImageURL,
		Status:          row.Status,
		Position:        row.Position,
		Total:           row.Total,
		ProgressPercent: row.ProgressPercent,
		LastUpdateAt:    row.LastUpdateAt,
		Rating:          normalizedRating(row.Rating, row.RatingScale),
	}
}

func normalizedRating(rating, scale *float64) *float64 {
	if rating == nil || scale == nil || *scale <= 0 {
		return nil
	}
	normalized := *rating / *scale * 10
	return &normalized
}
