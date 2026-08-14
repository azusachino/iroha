package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
	"github.com/go-chi/chi/v5"
)

type sleepListResponse struct {
	Items      []sleepResponse `json:"items"`
	NextCursor *string         `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}

type sleepOverviewResponse struct {
	SessionCount      int     `json:"session_count"`
	MainSleepCount    int     `json:"main_sleep_count"`
	AverageAsleepS    float64 `json:"average_asleep_s"`
	AverageEfficiency float64 `json:"average_efficiency"`
}

type sleepResponse struct {
	ID             string    `json:"id"`
	WakeDate       string    `json:"wake_date"`
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

type sleepAggregateResponse struct {
	Granularity string                         `json:"granularity"`
	Buckets     []sleepAggregateBucketResponse `json:"buckets"`
}

type sleepAggregateBucketResponse struct {
	Period            string  `json:"period"`
	SessionCount      int     `json:"session_count"`
	MainSleepCount    int     `json:"main_sleep_count"`
	NapCount          int     `json:"nap_count"`
	ObservedWakeDates int     `json:"observed_wake_dates"`
	AverageAsleepS    float64 `json:"average_asleep_s"`
	AverageTimeInBedS float64 `json:"average_time_in_bed_s"`
	AverageEfficiency float64 `json:"average_efficiency"`
	CoreS             int     `json:"core_s"`
	DeepS             int     `json:"deep_s"`
	RemS              int     `json:"rem_s"`
	AwakeS            int     `json:"awake_s"`
	UnspecifiedS      int     `json:"unspecified_s"`
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

func (s *Server) handleSleepOverview(w http.ResponseWriter, r *http.Request) {
	recentLimit := 30
	if value := r.URL.Query().Get("recent"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "invalid recent")
			return
		}
		recentLimit = parsed
	}
	overview, err := s.deps.SleepService.Overview(recentLimit)
	if err != nil {
		s.deps.Logger.Error("sleep overview", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load sleep overview")
		return
	}
	writeJSON(w, http.StatusOK, sleepOverviewResponse{
		SessionCount:      overview.SessionCount,
		MainSleepCount:    overview.MainSleepCount,
		AverageAsleepS:    overview.AverageAsleepS,
		AverageEfficiency: overview.AverageEfficiency,
	})
}

func (s *Server) handleGetSleep(w http.ResponseWriter, r *http.Request) {
	session, found, err := s.deps.SleepService.Get(chi.URLParam(r, "sleepId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid sleep id")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "sleep session not found")
		return
	}
	writeJSON(w, http.StatusOK, toSleepResponse(session))
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

func (s *Server) handleSleepAggregates(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseSleepAggregateFilters(w, r)
	if !ok {
		return
	}
	buckets, err := s.deps.SleepService.Aggregates(filters)
	if err != nil {
		s.deps.Logger.Error("aggregate sleep sessions", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to aggregate sleep sessions")
		return
	}
	response := sleepAggregateResponse{Granularity: filters.Granularity, Buckets: make([]sleepAggregateBucketResponse, 0, len(buckets))}
	for _, bucket := range buckets {
		response.Buckets = append(response.Buckets, sleepAggregateBucketResponse{
			Period:            formatAggregatePeriod(bucket.Period, filters.Granularity),
			SessionCount:      bucket.SessionCount,
			MainSleepCount:    bucket.MainSleepCount,
			NapCount:          bucket.NapCount,
			ObservedWakeDates: bucket.ObservedWakeDates,
			AverageAsleepS:    bucket.AverageAsleepS,
			AverageTimeInBedS: bucket.AverageTimeInBedS,
			AverageEfficiency: bucket.AverageEfficiency,
			CoreS:             bucket.CoreS,
			DeepS:             bucket.DeepS,
			RemS:              bucket.RemS,
			AwakeS:            bucket.AwakeS,
			UnspecifiedS:      bucket.UnspecifiedS,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func parseSleepFilters(w http.ResponseWriter, r *http.Request) (sleep.ListFilters, bool) {
	query := r.URL.Query()
	limit, ok := parsePageLimit(w, r)
	if !ok {
		return sleep.ListFilters{}, false
	}
	filters := sleep.ListFilters{Limit: limit}
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

func parseSleepAggregateFilters(w http.ResponseWriter, r *http.Request) (sleep.AggregateFilters, bool) {
	query := r.URL.Query()
	granularity := query.Get("granularity")
	if granularity == "" {
		granularity = "month"
	}
	if granularity != "month" && granularity != "year" && granularity != "lifetime" {
		writeError(w, http.StatusBadRequest, "invalid granularity")
		return sleep.AggregateFilters{}, false
	}
	filters := sleep.AggregateFilters{Granularity: granularity}
	for key, destination := range map[string]**time.Time{"from": &filters.From, "to": &filters.To} {
		if value := query.Get(key); value != "" {
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid "+key)
				return sleep.AggregateFilters{}, false
			}
			*destination = &parsed
		}
	}
	return filters, true
}

func toSleepResponse(session models.SleepSession) sleepResponse {
	return sleepResponse{
		ID:             ids.Encode(ids.SleepPrefix, session.ID),
		WakeDate:       formatCalendarDate(session.WakeDate),
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
