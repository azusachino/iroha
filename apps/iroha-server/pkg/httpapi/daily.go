package httpapi

import (
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
)

type dailyListResponse struct {
	Items      []dailyResponse `json:"items"`
	NextCursor *string         `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}

type dailyResponse struct {
	ID              string             `json:"id"`
	Day             string             `json:"day"`
	Ring            *dailyRingResponse `json:"ring"`
	Steps           *float64           `json:"steps,omitempty"`
	DistanceKM      *float64           `json:"distance_km,omitempty"`
	Flights         *float64           `json:"flights,omitempty"`
	RestingHR       *float64           `json:"resting_hr,omitempty"`
	WalkingHRAvg    *float64           `json:"walking_hr_avg,omitempty"`
	HRVSDNN         *float64           `json:"hrv_sdnn,omitempty"`
	SpO2Avg         *float64           `json:"spo2_avg,omitempty"`
	SpO2Min         *float64           `json:"spo2_min,omitempty"`
	RespiratoryRate *float64           `json:"respiratory_rate,omitempty"`
	VO2Max          *float64           `json:"vo2max,omitempty"`
	BodyMassKG      *float64           `json:"body_mass_kg,omitempty"`
	Source          string             `json:"source"`
	FirstRawFileID  string             `json:"first_raw_file_id"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type dailyRingResponse struct {
	MoveKcal        float64 `json:"move_kcal"`
	MoveGoalKcal    float64 `json:"move_goal_kcal"`
	ExerciseMin     float64 `json:"exercise_min"`
	ExerciseGoalMin float64 `json:"exercise_goal_min"`
	StandHours      float64 `json:"stand_hours"`
	StandGoalHours  float64 `json:"stand_goal_hours"`
}

func (s *Server) handleListDaily(w http.ResponseWriter, r *http.Request) {
	filters, ok := s.parseDailyFilters(w, r)
	if !ok {
		return
	}
	page, err := s.deps.DailyService.List(filters)
	if err != nil {
		s.deps.Logger.Error("list daily activity", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list daily activity")
		return
	}
	items := make([]dailyResponse, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, toDailyResponse(row))
	}
	response := dailyListResponse{Items: items, HasMore: page.HasMore}
	if page.NextCursor != nil {
		cursor := daily.EncodeCursor(*page.NextCursor)
		response.NextCursor = &cursor
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleDailyDates(w http.ResponseWriter, r *http.Request) {
	timezone, err := scopeLocation(r.URL.Query(), s.deps.Config.Server.Timezone)
	if err != nil {
		writeReadScopeError(w, err)
		return
	}
	dates, err := s.deps.DailyService.Dates(timezone.String())
	if err != nil {
		s.deps.Logger.Error("list daily dates", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list daily dates")
		return
	}
	response := make([]string, 0, len(dates))
	for _, date := range dates {
		response = append(response, formatCalendarDate(date))
	}
	writeJSON(w, http.StatusOK, response)
}

type dailyAggregateResponse struct {
	Granularity string                         `json:"granularity"`
	Buckets     []dailyAggregateBucketResponse `json:"buckets"`
}

type dailyAggregateBucketResponse struct {
	Period         string                         `json:"period"`
	Days           int                            `json:"days"`
	MoveKcalAvg    float64                        `json:"move_kcal_avg"`
	ExerciseMinAvg float64                        `json:"exercise_min_avg"`
	StandHoursAvg  float64                        `json:"stand_hours_avg"`
	MoveClosedPct  float64                        `json:"move_closed_pct"`
	Metrics        []dailyMetricAggregateResponse `json:"metrics"`
}

type dailyMetricAggregateResponse struct {
	Metric       string  `json:"metric"`
	Value        float64 `json:"value"`
	Unit         string  `json:"unit"`
	ObservedDays int     `json:"observed_days"`
}

func (s *Server) handleDailyAggregates(w http.ResponseWriter, r *http.Request) {
	filters, ok := s.parseDailyAggregateFilters(w, r)
	if !ok {
		return
	}
	buckets, err := s.deps.DailyService.Aggregates(filters)
	if err != nil {
		s.deps.Logger.Error("aggregate daily activity", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to aggregate daily activity")
		return
	}
	response := dailyAggregateResponse{Granularity: filters.Granularity, Buckets: make([]dailyAggregateBucketResponse, 0, len(buckets))}
	for _, bucket := range buckets {
		response.Buckets = append(response.Buckets, dailyAggregateBucketResponse{
			Period:         formatAggregatePeriod(bucket.Period, filters.Granularity),
			Days:           bucket.Days,
			MoveKcalAvg:    bucket.MoveKcalAvg,
			ExerciseMinAvg: bucket.ExerciseMinAvg,
			StandHoursAvg:  bucket.StandHoursAvg,
			MoveClosedPct:  bucket.MoveClosedPct,
			Metrics:        toDailyMetricAggregateResponses(bucket.Metrics),
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func toDailyMetricAggregateResponses(metrics []daily.MetricAggregate) []dailyMetricAggregateResponse {
	response := make([]dailyMetricAggregateResponse, 0, len(metrics))
	for _, metric := range metrics {
		response = append(response, dailyMetricAggregateResponse{
			Metric: metric.Metric, Value: metric.Value, Unit: metric.Unit, ObservedDays: metric.ObservedDays,
		})
	}
	return response
}

func parseDailyAggregateFilters(w http.ResponseWriter, r *http.Request) (daily.AggregateFilters, bool) {
	query := r.URL.Query()
	granularity := query.Get("granularity")
	if granularity == "" {
		granularity = "month"
	}
	if granularity != "month" && granularity != "year" {
		writeError(w, http.StatusBadRequest, "invalid granularity")
		return daily.AggregateFilters{}, false
	}
	filters := daily.AggregateFilters{Granularity: granularity}
	for key, destination := range map[string]**time.Time{"from": &filters.From, "to": &filters.To} {
		if value := query.Get(key); value != "" {
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid "+key)
				return daily.AggregateFilters{}, false
			}
			*destination = &parsed
		}
	}
	return filters, true
}

func parseDailyFilters(w http.ResponseWriter, r *http.Request) (daily.ListFilters, bool) {
	query := r.URL.Query()
	limit, ok := parsePageLimit(w, r)
	if !ok {
		return daily.ListFilters{}, false
	}
	filters := daily.ListFilters{Limit: limit}
	for key, destination := range map[string]**time.Time{"from": &filters.From, "to": &filters.To} {
		if value := query.Get(key); value != "" {
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid "+key)
				return daily.ListFilters{}, false
			}
			*destination = &parsed
		}
	}
	if value := query.Get("cursor"); value != "" {
		cursor, err := daily.DecodeCursor(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return daily.ListFilters{}, false
		}
		filters.Cursor = &cursor
	}
	return filters, true
}

func toDailyResponse(row daily.Row) dailyResponse {
	summary := row.DailySummary
	var ring *dailyRingResponse
	if row.RingPresent {
		ring = &dailyRingResponse{
			MoveKcal: summary.MoveKcal, MoveGoalKcal: summary.MoveGoalKcal,
			ExerciseMin: summary.ExerciseMin, ExerciseGoalMin: summary.ExerciseGoalMin,
			StandHours: summary.StandHours, StandGoalHours: summary.StandGoalHours,
		}
	}
	return dailyResponse{
		ID:              ids.Encode(ids.DailySummaryPrefix, summary.ID),
		Day:             formatCalendarDate(summary.Day),
		Ring:            ring,
		Steps:           row.Steps,
		DistanceKM:      row.DistanceKM,
		Flights:         row.Flights,
		RestingHR:       row.RestingHR,
		WalkingHRAvg:    row.WalkingHRAvg,
		HRVSDNN:         row.HRVSDNN,
		SpO2Avg:         row.SpO2Avg,
		SpO2Min:         row.SpO2Min,
		RespiratoryRate: row.RespiratoryRate,
		VO2Max:          row.VO2Max,
		BodyMassKG:      row.BodyMassKG,
		Source:          summary.Source,
		FirstRawFileID:  ids.Encode(ids.RawFilePrefix, summary.FirstRawFileID),
		CreatedAt:       summary.CreatedAt,
		UpdatedAt:       summary.UpdatedAt,
	}
}

func formatCalendarDate(value time.Time) string {
	return value.Format("2006-01-02")
}

func formatAggregatePeriod(value time.Time, granularity string) string {
	if granularity == "lifetime" {
		return "lifetime"
	}
	if granularity == "year" {
		return value.Format("2006")
	}
	return value.Format("2006-01")
}
