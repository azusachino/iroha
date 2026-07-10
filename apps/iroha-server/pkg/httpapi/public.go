package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
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
	yearParam := r.URL.Query().Get("year")
	sportParam := r.URL.Query().Get("sport")
	cacheKey := fmt.Sprintf("public:summary:v1:%s:%s", yearParam, sportParam)

	ttl := publicCacheTTL
	currentYear := time.Now().Format("2006")
	if yearParam == "" || yearParam == currentYear {
		ttl = 1 * time.Minute // Active periods have short TTL
	}

	summary, err := cache.GetOrLoad(
		r.Context(), s.deps.Cache, cacheKey, ttl,
		func() (activities.Summary, error) { return s.deps.ActivityService.Summary(yearParam, sportParam) },
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
	Year      string `json:"year"`
	City      string `json:"city"`
}

func (s *Server) handlePublicRoutes(w http.ResponseWriter, r *http.Request) {
	response, err := cache.GetOrLoad(
		r.Context(), s.deps.Cache, "public:routes:v2", 5*time.Minute,
		func() (geoJSONFeatureCollection, error) {
			lines, err := s.deps.ActivityService.RouteLines()
			if err != nil {
				return geoJSONFeatureCollection{}, err
			}

			features := make([]geoJSONFeature, 0, len(lines))
			for _, line := range lines {
				city := "Unknown"
				if len(line.Points) > 0 {
					lon := line.Points[0][0]
					lat := line.Points[0][1]
					key := fmt.Sprintf("geocode:v1:%.2f:%.2f", lat, lon)
					if val, ok := cache.Get[string](r.Context(), s.deps.Cache, key); ok {
						city = val
					} else {
						s.enqueueBackgroundGeocode(lat, lon)
					}
				}

				features = append(features, geoJSONFeature{
					Type:       "Feature",
					Geometry:   geoJSONLineString{Type: "LineString", Coordinates: line.Points},
					Properties: routeLineProps{SportType: line.SportType, Year: line.Year, City: city},
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

func (s *Server) handlePublicGeocode(w http.ResponseWriter, r *http.Request) {
	lat := r.URL.Query().Get("lat")
	lon := r.URL.Query().Get("lon")
	if lat == "" || lon == "" {
		writeError(w, http.StatusBadRequest, "lat and lon are required")
		return
	}

	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?lat=%s&lon=%s&format=json&zoom=10", lat, lon)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create request")
		return
	}
	req.Header.Set("User-Agent", "Iroha-Fitness-Cockpit/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.deps.Logger.Error("geocode proxy failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to query geocoder")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		writeError(w, resp.StatusCode, "geocoder returned error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, resp.Body)
}

var (
	geocodePendingMu sync.Mutex
	geocodePending   = make(map[string]bool)
	geocodeChan      = make(chan [2]float64, 1000)
	geocodeOnce      sync.Once
)

func (s *Server) enqueueBackgroundGeocode(lat, lon float64) {
	geocodeOnce.Do(func() {
		go s.startGeocodeWorker()
	})

	key := fmt.Sprintf("geocode:v1:%.2f:%.2f", lat, lon)

	geocodePendingMu.Lock()
	if geocodePending[key] {
		geocodePendingMu.Unlock()
		return
	}
	geocodePending[key] = true
	geocodePendingMu.Unlock()

	select {
	case geocodeChan <- [2]float64{lat, lon}:
	default:
		geocodePendingMu.Lock()
		delete(geocodePending, key)
		geocodePendingMu.Unlock()
	}
}

func (s *Server) startGeocodeWorker() {
	for coord := range geocodeChan {
		lat, lon := coord[0], coord[1]
		key := fmt.Sprintf("geocode:v1:%.2f:%.2f", lat, lon)

		// Respect the rate limit: sleep 1 second between requests
		time.Sleep(1 * time.Second)

		url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?lat=%.6f&lon=%.6f&format=json&zoom=10", lat, lon)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			s.releasePending(key)
			continue
		}
		req.Header.Set("User-Agent", "Iroha-Fitness-Cockpit/1.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			s.deps.Logger.Error("background geocode failed", "error", err, "key", key)
			s.releasePending(key)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			s.releasePending(key)
			continue
		}

		var data struct {
			Address map[string]string `json:"address"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			_ = resp.Body.Close()
			s.releasePending(key)
			continue
		}
		_ = resp.Body.Close()

		city := data.Address["city"]
		if city == "" {
			city = data.Address["town"]
		}
		if city == "" {
			city = data.Address["village"]
		}
		if city == "" {
			city = data.Address["city_district"]
		}
		if city == "" {
			city = data.Address["county"]
		}
		if city == "" {
			city = data.Address["state"]
		}
		if city == "" {
			city = "Unknown"
		}

		cache.Set(context.Background(), s.deps.Cache, key, 365*24*time.Hour, city)

		// Invalidate public:routes cache key to trigger reload
		_ = s.deps.Cache.DeletePattern(context.Background(), "public:routes:*")

		s.releasePending(key)
	}
}

func (s *Server) releasePending(key string) {
	geocodePendingMu.Lock()
	delete(geocodePending, key)
	geocodePendingMu.Unlock()
}
