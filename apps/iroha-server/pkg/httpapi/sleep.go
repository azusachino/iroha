package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
	"github.com/go-chi/chi/v5"
)

type sleepListResponse struct {
	Items      []sleepResponse `json:"items"`
	NextCursor *string         `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}

type sleepResponse struct {
	ID             string    `json:"id"`
	WakeDate       time.Time `json:"wake_date"`
	StartedAt      time.Time `json:"started_at"`
	EndedAt        time.Time `json:"ended_at"`
	TimeInBedS     int       `json:"time_in_bed_s"`
	AsleepS        int       `json:"asleep_s"`
	Efficiency     float64   `json:"efficiency"`
	IsMainSleep    bool      `json:"is_main_sleep"`
	CoreS          int       `json:"core_s"`
	DeepS          int       `json:"deep_s"`
	RemS           int       `json:"rem_s"`
	AwakeS         int       `json:"awake_s"`
	UnspecifiedS   int       `json:"unspecified_s"`
	Source         string    `json:"source"`
	FirstRawFileID string    `json:"first_raw_file_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type sleepSegmentResponse struct {
	ID        string    `json:"id"`
	Stage     string    `json:"stage"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Seq       int       `json:"seq"`
}

func (s *Server) handleListSleep(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseSleepFilters(w, r)
	if !ok {
		return
	}
	page, err := s.deps.SleepService.List(filters)
	if err != nil {
		s.deps.Logger.Error("list sleep sessions", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list sleep sessions")
		return
	}
	items := make([]sleepResponse, 0, len(page.Items))
	for _, session := range page.Items {
		items = append(items, toSleepResponse(session))
	}
	response := sleepListResponse{Items: items, HasMore: page.HasMore}
	if page.NextCursor != nil {
		cursor := sleep.EncodeCursor(*page.NextCursor)
		response.NextCursor = &cursor
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetSleepSegments(w http.ResponseWriter, r *http.Request) {
	segments, found, err := s.deps.SleepService.Segments(chi.URLParam(r, "sleepId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid sleep id")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "sleep session not found")
		return
	}
	response := make([]sleepSegmentResponse, 0, len(segments))
	for _, segment := range segments {
		response = append(response, sleepSegmentResponse{
			ID:        ids.Encode(ids.SleepSegmentPrefix, segment.ID),
			Stage:     segment.Stage,
			StartedAt: segment.StartedAt,
			EndedAt:   segment.EndedAt,
			Seq:       segment.Seq,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func parseSleepFilters(w http.ResponseWriter, r *http.Request) (sleep.ListFilters, bool) {
	query := r.URL.Query()
	filters := sleep.ListFilters{}
	for key, destination := range map[string]**time.Time{"from": &filters.From, "to": &filters.To} {
		if value := query.Get(key); value != "" {
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid "+key)
				return sleep.ListFilters{}, false
			}
			*destination = &parsed
		}
	}
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return sleep.ListFilters{}, false
		}
		filters.Limit = limit
	}
	if value := query.Get("cursor"); value != "" {
		cursor, err := sleep.DecodeCursor(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return sleep.ListFilters{}, false
		}
		filters.Cursor = &cursor
	}
	return filters, true
}

func toSleepResponse(session models.SleepSession) sleepResponse {
	return sleepResponse{
		ID:             ids.Encode(ids.SleepPrefix, session.ID),
		WakeDate:       session.WakeDate,
		StartedAt:      session.StartedAt,
		EndedAt:        session.EndedAt,
		TimeInBedS:     session.TimeInBedS,
		AsleepS:        session.AsleepS,
		Efficiency:     session.Efficiency,
		IsMainSleep:    session.IsMainSleep,
		CoreS:          session.CoreS,
		DeepS:          session.DeepS,
		RemS:           session.RemS,
		AwakeS:         session.AwakeS,
		UnspecifiedS:   session.UnspecifiedS,
		Source:         session.Source,
		FirstRawFileID: ids.Encode(ids.RawFilePrefix, session.FirstRawFileID),
		CreatedAt:      session.CreatedAt,
		UpdatedAt:      session.UpdatedAt,
	}
}
