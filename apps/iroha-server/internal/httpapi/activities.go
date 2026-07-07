package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/internal/activities"
	"github.com/azusachino/iroha/apps/iroha-server/internal/ids"
	"github.com/azusachino/iroha/apps/iroha-server/internal/models"
	"github.com/go-chi/chi/v5"
)

type activityResponse struct {
	ID               string     `json:"id"`
	SportType        string     `json:"sport_type"`
	Title            string     `json:"title"`
	StartedAt        time.Time  `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at,omitempty"`
	Timezone         string     `json:"timezone"`
	DistanceM        *float64   `json:"distance_m,omitempty"`
	DurationS        *int       `json:"duration_s,omitempty"`
	MovingTimeS      *int       `json:"moving_time_s,omitempty"`
	ElevationGainM   *float64   `json:"elevation_gain_m,omitempty"`
	AvgHR            *int       `json:"avg_hr,omitempty"`
	MaxHR            *int       `json:"max_hr,omitempty"`
	AvgPaceSPerKM    *float64   `json:"avg_pace_s_per_km,omitempty"`
	SourceKind       string     `json:"source_kind"`
	SourceActivityID string     `json:"source_activity_id,omitempty"`
	FirstRawFileID   string     `json:"first_raw_file_id"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type routePointResponse struct {
	Seq        int        `json:"seq"`
	Ts         *time.Time `json:"ts,omitempty"`
	Lat        float64    `json:"lat"`
	Lon        float64    `json:"lon"`
	ElevationM *float64   `json:"elevation_m,omitempty"`
	DistanceM  *float64   `json:"distance_m,omitempty"`
	SpeedMPS   *float64   `json:"speed_mps,omitempty"`
	HeartRate  *int       `json:"heart_rate,omitempty"`
}

type activitySamplingResponse struct {
	ID           string    `json:"id"`
	SamplingType string    `json:"sampling_type"`
	Ts           time.Time `json:"ts"`
	Value        float64   `json:"value"`
	Unit         string    `json:"unit"`
}

type activityLapResponse struct {
	ID            string     `json:"id"`
	LapNo         int        `json:"lap_no"`
	StartTs       *time.Time `json:"start_ts,omitempty"`
	EndTs         *time.Time `json:"end_ts,omitempty"`
	DistanceM     *float64   `json:"distance_m,omitempty"`
	DurationS     *int       `json:"duration_s,omitempty"`
	AvgHR         *int       `json:"avg_hr,omitempty"`
	AvgPaceSPerKM *float64   `json:"avg_pace_s_per_km,omitempty"`
}

func (s *Server) handleListActivities(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseActivityFilters(w, r)
	if !ok {
		return
	}

	rows, err := s.deps.ActivityService.List(filters)
	if err != nil {
		s.deps.Logger.Error("list activities", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list activities")
		return
	}

	response := make([]activityResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, toActivityResponse(row))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetActivity(w http.ResponseWriter, r *http.Request) {
	activity, found, err := s.deps.ActivityService.Get(chi.URLParam(r, "activityId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid activity id")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "activity not found")
		return
	}
	writeJSON(w, http.StatusOK, toActivityResponse(activity))
}

func (s *Server) handleGetActivityRoute(w http.ResponseWriter, r *http.Request) {
	points, found, err := s.deps.ActivityService.Route(chi.URLParam(r, "activityId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid activity id")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "activity not found")
		return
	}

	response := make([]routePointResponse, 0, len(points))
	for _, point := range points {
		response = append(response, routePointResponse{
			Seq:        point.Seq,
			Ts:         point.Ts,
			Lat:        point.Lat,
			Lon:        point.Lon,
			ElevationM: point.ElevationM,
			DistanceM:  point.DistanceM,
			SpeedMPS:   point.SpeedMPS,
			HeartRate:  point.HeartRate,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetActivitySamplings(w http.ResponseWriter, r *http.Request) {
	samplings, found, err := s.deps.ActivityService.Samplings(chi.URLParam(r, "activityId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid activity id")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "activity not found")
		return
	}

	response := make([]activitySamplingResponse, 0, len(samplings))
	for _, sampling := range samplings {
		response = append(response, activitySamplingResponse{
			ID:           ids.Encode("sampling", sampling.ID),
			SamplingType: sampling.SamplingType,
			Ts:           sampling.Ts,
			Value:        sampling.Value,
			Unit:         sampling.Unit,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetActivityLaps(w http.ResponseWriter, r *http.Request) {
	laps, found, err := s.deps.ActivityService.Laps(chi.URLParam(r, "activityId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid activity id")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "activity not found")
		return
	}

	response := make([]activityLapResponse, 0, len(laps))
	for _, lap := range laps {
		response = append(response, activityLapResponse{
			ID:            ids.Encode("lap", lap.ID),
			LapNo:         lap.LapNo,
			StartTs:       lap.StartTs,
			EndTs:         lap.EndTs,
			DistanceM:     lap.DistanceM,
			DurationS:     lap.DurationS,
			AvgHR:         lap.AvgHR,
			AvgPaceSPerKM: lap.AvgPaceSPerKM,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func parseActivityFilters(w http.ResponseWriter, r *http.Request) (activities.ListFilters, bool) {
	query := r.URL.Query()
	filters := activities.ListFilters{SportType: query.Get("sport_type")}

	if value := query.Get("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return activities.ListFilters{}, false
		}
		filters.Limit = limit
	}
	if value := query.Get("started_from"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid started_from")
			return activities.ListFilters{}, false
		}
		filters.StartedFrom = &parsed
	}
	if value := query.Get("started_to"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid started_to")
			return activities.ListFilters{}, false
		}
		filters.StartedTo = &parsed
	}

	return filters, true
}

func toActivityResponse(activity models.Activity) activityResponse {
	return activityResponse{
		ID:               ids.Encode(ids.ActivityPrefix, activity.ID),
		SportType:        activity.SportType,
		Title:            activity.Title,
		StartedAt:        activity.StartedAt,
		EndedAt:          activity.EndedAt,
		Timezone:         activity.Timezone,
		DistanceM:        activity.DistanceM,
		DurationS:        activity.DurationS,
		MovingTimeS:      activity.MovingTimeS,
		ElevationGainM:   activity.ElevationGainM,
		AvgHR:            activity.AvgHR,
		MaxHR:            activity.MaxHR,
		AvgPaceSPerKM:    activity.AvgPaceSPerKM,
		SourceKind:       activity.SourceKind,
		SourceActivityID: activity.SourceActivityID,
		FirstRawFileID:   ids.Encode(ids.RawFilePrefix, activity.FirstRawFileID),
		CreatedAt:        activity.CreatedAt,
		UpdatedAt:        activity.UpdatedAt,
	}
}
