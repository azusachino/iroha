package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/ids"
)

type dailyListResponse struct {
	Items      []dailyResponse `json:"items"`
	NextCursor *string         `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}

type dailyResponse struct {
	ID              string    `json:"id"`
	Day             time.Time `json:"day"`
	MoveKcal        float64   `json:"move_kcal"`
	MoveGoalKcal    float64   `json:"move_goal_kcal"`
	ExerciseMin     float64   `json:"exercise_min"`
	ExerciseGoalMin float64   `json:"exercise_goal_min"`
	StandHours      float64   `json:"stand_hours"`
	StandGoalHours  float64   `json:"stand_goal_hours"`
	Steps           *float64  `json:"steps,omitempty"`
	DistanceKM      *float64  `json:"distance_km,omitempty"`
	Flights         *float64  `json:"flights,omitempty"`
	Source          string    `json:"source"`
	FirstRawFileID  string    `json:"first_raw_file_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (s *Server) handleListDaily(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseDailyFilters(w, r)
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

func parseDailyFilters(w http.ResponseWriter, r *http.Request) (daily.ListFilters, bool) {
	query := r.URL.Query()
	filters := daily.ListFilters{}
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
	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return daily.ListFilters{}, false
		}
		filters.Limit = limit
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
	return dailyResponse{
		ID:              ids.Encode(ids.DailySummaryPrefix, summary.ID),
		Day:             summary.Day,
		MoveKcal:        summary.MoveKcal,
		MoveGoalKcal:    summary.MoveGoalKcal,
		ExerciseMin:     summary.ExerciseMin,
		ExerciseGoalMin: summary.ExerciseGoalMin,
		StandHours:      summary.StandHours,
		StandGoalHours:  summary.StandGoalHours,
		Steps:           row.Steps,
		DistanceKM:      row.DistanceKM,
		Flights:         row.Flights,
		Source:          summary.Source,
		FirstRawFileID:  ids.Encode(ids.RawFilePrefix, summary.FirstRawFileID),
		CreatedAt:       summary.CreatedAt,
		UpdatedAt:       summary.UpdatedAt,
	}
}
