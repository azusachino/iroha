package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/publicexport"
	"github.com/go-chi/chi/v5"
)

type activityListResponse struct {
	Items      []activityResponse `json:"items"`
	NextCursor *string            `json:"next_cursor"`
	HasMore    bool               `json:"has_more"`
}

type activityOverviewResponse struct {
	Summary       activities.Summary     `json:"summary"`
	ActiveDays    []activities.ActiveDay `json:"active_days"`
	Recent        []activityResponse     `json:"recent"`
	CurrentStreak int                    `json:"current_streak"`
}

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

	page, err := s.deps.ActivityService.List(filters)
	if err != nil {
		s.deps.Logger.Error("list activities", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list activities")
		return
	}

	items := make([]activityResponse, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, toActivityResponse(row))
	}

	response := activityListResponse{Items: items, HasMore: page.HasMore}
	if page.NextCursor != nil {
		cursor := activities.EncodeCursor(*page.NextCursor)
		response.NextCursor = &cursor
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
	// Optional ?type=heart_rate,running_power narrows the stream to the requested
	// sampling_types; omit for the full set.
	var types []string
	if raw := r.URL.Query().Get("type"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				types = append(types, t)
			}
		}
	}
	samplings, found, err := s.deps.ActivityService.Samplings(chi.URLParam(r, "activityId"), types...)
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

// handleActivitySummary and handleActivityRoutes back the dashboard/activities
// aggregate widgets. Both reuse publicexport (originally the sanitized
// /public/v1 query logic) since the aggregates it builds carry no private
// fields to begin with — there's nothing left to sanitize away.
func (s *Server) handleActivitySummary(w http.ResponseWriter, r *http.Request) {
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = s.deps.Config.Server.Timezone
	}
	summary, err := s.deps.ActivityService.SummaryInTimezone(r.URL.Query().Get("year"), r.URL.Query().Get("sport"), timezone)
	if err != nil {
		if strings.Contains(err.Error(), "load timezone") {
			writeError(w, http.StatusBadRequest, "invalid timezone")
			return
		}
		s.deps.Logger.Error("activity summary", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleActivityOverview(w http.ResponseWriter, r *http.Request) {
	timezone := r.URL.Query().Get("timezone")
	if timezone == "" {
		timezone = s.deps.Config.Server.Timezone
	}
	recentLimit := 5
	if value := r.URL.Query().Get("recent"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "invalid recent")
			return
		}
		recentLimit = parsed
	}
	overview, err := s.deps.ActivityService.Overview(timezone, recentLimit)
	if err != nil {
		if strings.Contains(err.Error(), "load timezone") {
			writeError(w, http.StatusBadRequest, "invalid timezone")
			return
		}
		s.deps.Logger.Error("activity overview", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load activity overview")
		return
	}
	recent := make([]activityResponse, 0, len(overview.Recent))
	for _, activity := range overview.Recent {
		recent = append(recent, toActivityResponse(activity))
	}
	writeJSON(w, http.StatusOK, activityOverviewResponse{
		Summary:       overview.Summary,
		ActiveDays:    overview.ActiveDays,
		Recent:        recent,
		CurrentStreak: overview.CurrentStreak,
	})
}

func (s *Server) handleActivityRoutes(w http.ResponseWriter, r *http.Request) {
	collection, err := publicexport.Routes(r.Context(), s.deps.ActivityService, s.deps.GeocodeService, true)
	if err != nil {
		s.deps.Logger.Error("activity routes", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load routes")
		return
	}
	writeJSON(w, http.StatusOK, collection)
}

func parseActivityFilters(w http.ResponseWriter, r *http.Request) (activities.ListFilters, bool) {
	query := r.URL.Query()
	limit, ok := parsePageLimit(w, r)
	if !ok {
		return activities.ListFilters{}, false
	}
	filters := activities.ListFilters{SportType: query.Get("sport_type"), Limit: limit}
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
	if value := query.Get("min_distance_m"); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid min_distance_m")
			return activities.ListFilters{}, false
		}
		filters.DistanceMinM = &parsed
	}
	if value := query.Get("max_distance_m"); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid max_distance_m")
			return activities.ListFilters{}, false
		}
		filters.DistanceMaxM = &parsed
	}
	if value := query.Get("cursor"); value != "" {
		cursor, err := activities.DecodeCursor(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid cursor")
			return activities.ListFilters{}, false
		}
		filters.Cursor = &cursor
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
