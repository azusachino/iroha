package httpapi

import (
	"net/http"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/cache"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/models"
)

// publicCacheTTL is how long public responses are cached in valkey. The page
// is a once-a-day rollup, so a day-long TTL is plenty.
const publicCacheTTL = 24 * time.Hour

// publicActivityResponse is the sanitized public view of an activity. It is a
// deliberate, separate type from activityResponse: it MUST NOT carry internal
// or source-linking fields (first_raw_file_id, source_activity_id, source_kind,
// internal timestamps). Do not replace this with the private DTO.
type publicActivityResponse struct {
	ID             string     `json:"id"`
	SportType      string     `json:"sport_type"`
	Title          string     `json:"title"`
	StartedAt      time.Time  `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	Timezone       string     `json:"timezone"`
	DistanceM      *float64   `json:"distance_m,omitempty"`
	DurationS      *int       `json:"duration_s,omitempty"`
	MovingTimeS    *int       `json:"moving_time_s,omitempty"`
	ElevationGainM *float64   `json:"elevation_gain_m,omitempty"`
	AvgHR          *int       `json:"avg_hr,omitempty"`
	MaxHR          *int       `json:"max_hr,omitempty"`
	AvgPaceSPerKM  *float64   `json:"avg_pace_s_per_km,omitempty"`
}

type publicActivityListResponse struct {
	Items      []publicActivityResponse `json:"items"`
	NextCursor *string                  `json:"next_cursor"`
	HasMore    bool                     `json:"has_more"`
}

func (s *Server) handlePublicSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := cache.GetOrLoad(
		r.Context(), s.deps.Cache, "public:summary:v1", publicCacheTTL,
		func() (activities.Summary, error) { return s.deps.ActivityService.Summary() },
	)
	if err != nil {
		s.deps.Logger.Error("public summary", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handlePublicActivities(w http.ResponseWriter, r *http.Request) {
	// Parse (and validate) filters before the cache lookup so bad input returns
	// a 400 and is never cached.
	filters, ok := parseActivityFilters(w, r)
	if !ok {
		return
	}

	key := "public:activities:v1:" + r.URL.Query().Encode()
	response, err := cache.GetOrLoad(
		r.Context(), s.deps.Cache, key, publicCacheTTL,
		func() (publicActivityListResponse, error) {
			page, err := s.deps.ActivityService.List(filters)
			if err != nil {
				return publicActivityListResponse{}, err
			}

			items := make([]publicActivityResponse, 0, len(page.Items))
			for _, row := range page.Items {
				items = append(items, toPublicActivityResponse(row))
			}
			out := publicActivityListResponse{Items: items, HasMore: page.HasMore}
			if page.NextCursor != nil {
				cursor := activities.EncodeCursor(*page.NextCursor)
				out.NextCursor = &cursor
			}
			return out, nil
		},
	)
	if err != nil {
		s.deps.Logger.Error("public activities", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list activities")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

// geoJSONFeatureCollection and geoJSONFeature are minimal GeoJSON types for
// the public routes response — just enough to describe a LineString per
// route, without pulling in a full GeoJSON dependency.
type geoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []geoJSONFeature `json:"features"`
}

type geoJSONFeature struct {
	Type       string            `json:"type"`
	Geometry   geoJSONLineString `json:"geometry"`
	Properties routeLineProps    `json:"properties"`
}

type geoJSONLineString struct {
	Type        string       `json:"type"`
	Coordinates [][2]float64 `json:"coordinates"`
}

type routeLineProps struct {
	SportType string `json:"sport_type"`
}

func (s *Server) handlePublicRoutes(w http.ResponseWriter, r *http.Request) {
	response, err := cache.GetOrLoad(
		r.Context(), s.deps.Cache, "public:routes:v1", publicCacheTTL,
		func() (geoJSONFeatureCollection, error) {
			lines, err := s.deps.ActivityService.RouteLines()
			if err != nil {
				return geoJSONFeatureCollection{}, err
			}

			features := make([]geoJSONFeature, 0, len(lines))
			for _, line := range lines {
				features = append(features, geoJSONFeature{
					Type:       "Feature",
					Geometry:   geoJSONLineString{Type: "LineString", Coordinates: line.Points},
					Properties: routeLineProps{SportType: line.SportType},
				})
			}
			return geoJSONFeatureCollection{Type: "FeatureCollection", Features: features}, nil
		},
	)
	if err != nil {
		s.deps.Logger.Error("public routes", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load routes")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func toPublicActivityResponse(activity models.Activity) publicActivityResponse {
	return publicActivityResponse{
		ID:             ids.Encode(ids.ActivityPrefix, activity.ID),
		SportType:      activity.SportType,
		Title:          activity.Title,
		StartedAt:      activity.StartedAt,
		EndedAt:        activity.EndedAt,
		Timezone:       activity.Timezone,
		DistanceM:      activity.DistanceM,
		DurationS:      activity.DurationS,
		MovingTimeS:    activity.MovingTimeS,
		ElevationGainM: activity.ElevationGainM,
		AvgHR:          activity.AvgHR,
		MaxHR:          activity.MaxHR,
		AvgPaceSPerKM:  activity.AvgPaceSPerKM,
	}
}
