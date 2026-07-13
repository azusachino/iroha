package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
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
