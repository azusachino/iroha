// Package publicexport builds the public-facing projection of activity data.
// The archive-wide view is sanitized; an explicit allowlist may additionally
// publish selected activity detail records. It has no HTTP or cache
// dependency: callers get plain data back and decide what to do with it.
package publicexport

import (
	"context"
	"fmt"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
)

// Activity is the sanitized public view of an activity. It is a deliberate,
// separate type from the private activity DTO: it MUST NOT carry internal or
// source-linking fields (first_raw_file_id, source_activity_id, source_kind,
// internal timestamps). Do not replace this with the private DTO.
//
// Title ships verbatim and is not covered by the route/geo sanitization
// below — it's freeform text, so avoid putting private notes in workout
// titles if this data is ever published.
type Activity struct {
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

// ActivityPage mirrors activities.Page, but with sanitized items and an
// already-encoded cursor token.
type ActivityPage struct {
	Items      []Activity `json:"items"`
	NextCursor *string    `json:"next_cursor"`
	HasMore    bool       `json:"has_more"`
}

// ToActivity sanitizes one activity for public consumption.
func ToActivity(activity models.Activity) Activity {
	return Activity{
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

// Activities returns one sanitized page of activities matching filters.
func Activities(svc *activities.Service, filters activities.ListFilters) (ActivityPage, error) {
	page, err := svc.List(filters)
	if err != nil {
		return ActivityPage{}, err
	}

	items := make([]Activity, 0, len(page.Items))
	for _, row := range page.Items {
		items = append(items, ToActivity(row))
	}
	out := ActivityPage{Items: items, HasMore: page.HasMore}
	if page.NextCursor != nil {
		cursor := activities.EncodeCursor(*page.NextCursor)
		out.NextCursor = &cursor
	}
	return out, nil
}

// Summary is a thin pass-through — activities.Summary is already an
// aggregate, so it carries nothing that needs sanitizing.
func Summary(svc *activities.Service, year, sport string) (activities.Summary, error) {
	return svc.Summary(year, sport)
}

// Meta carries when a static snapshot was generated, so the public site can
// show a freshness indicator instead of implying its data is live.
type Meta struct {
	GeneratedAt    time.Time `json:"generated_at"`
	RoutesIncluded bool      `json:"routes_included"`
	ActivityCount  int       `json:"activity_count"`
}

// RouteFeatureCollection and RouteFeature are minimal GeoJSON types for the
// public route lines — just enough to describe a LineString per route.
type RouteFeatureCollection struct {
	Type     string         `json:"type"`
	Features []RouteFeature `json:"features"`
}

type RouteFeature struct {
	Type       string            `json:"type"`
	Geometry   RouteGeometry     `json:"geometry"`
	Properties RouteFeatureProps `json:"properties"`
}

type RouteGeometry struct {
	Type        string       `json:"type"`
	Coordinates [][2]float64 `json:"coordinates"`
}

type RouteFeatureProps struct {
	ActivityID string `json:"activity_id"`
	SportType  string `json:"sport_type"`
	Year       string `json:"year"`
	City       string `json:"city"`
	CityStatus string `json:"city_status"`
}

// Routes returns every activity route line, trimmed and privacy-masked by
// activities.Service.RouteLines, with a best-effort city label resolved from
// the existing geocode cache. EnqueueRefresh is the only place that ever
// warms the geocode cache, so refreshOnMiss must be true for the live
// /api/v1/activities/routes handler (it's how city labels resolve at all over
// time); the read-only iroha-export-public CLI passes false, since a static
// export run shouldn't be triggering new background lookups.
func Routes(ctx context.Context, activitySvc *activities.Service, geocodeSvc *geocode.Service, refreshOnMiss bool) (RouteFeatureCollection, error) {
	lines, err := activitySvc.RouteLines()
	if err != nil {
		return RouteFeatureCollection{}, fmt.Errorf("route lines: %w", err)
	}

	features := make([]RouteFeature, 0, len(lines))
	for _, line := range lines {
		city := "Unknown"
		cityStatus := "pending"
		if len(line.Points) > 0 && geocodeSvc != nil {
			lon := line.Points[0][0]
			lat := line.Points[0][1]
			if val, ok, err := geocodeSvc.LookupCity(ctx, lat, lon); err == nil && ok {
				city = val
				cityStatus = "resolved"
			} else if refreshOnMiss {
				_ = geocodeSvc.EnqueueRefresh(ctx, lat, lon)
			}
		}

		features = append(features, RouteFeature{
			Type:     "Feature",
			Geometry: RouteGeometry{Type: "LineString", Coordinates: line.Points},
			Properties: RouteFeatureProps{
				ActivityID: ids.Encode(ids.ActivityPrefix, line.ActivityID),
				SportType:  line.SportType,
				Year:       line.Year,
				City:       city,
				CityStatus: cityStatus,
			},
		})
	}
	return RouteFeatureCollection{Type: "FeatureCollection", Features: features}, nil
}
