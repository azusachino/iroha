package publicexport

import (
	"fmt"
	"strings"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
)

// Validate checks the sanitized export for schema and privacy invariants
// immediately before it is written to disk and published. It is a
// regression gate: a future change to Activity, Summary, or route
// sanitization that reintroduces a private field or an out-of-range value
// should fail here instead of silently reaching the public site.
func Validate(summary activities.Summary, activityList []Activity, routes RouteFeatureCollection) error {
	if err := validateSummary(summary); err != nil {
		return fmt.Errorf("summary: %w", err)
	}
	for _, activity := range activityList {
		if err := validateActivity(activity); err != nil {
			return fmt.Errorf("activity %s: %w", activity.ID, err)
		}
	}
	for i, feature := range routes.Features {
		if err := validateRouteFeature(feature); err != nil {
			return fmt.Errorf("route feature %d: %w", i, err)
		}
	}
	return nil
}

func validateSummary(summary activities.Summary) error {
	if summary.Totals.ActivityCount < 0 || summary.Totals.DistanceM < 0 ||
		summary.Totals.DistanceKnownCount < 0 || summary.Totals.DistanceUnknownCount < 0 ||
		summary.Totals.DurationS < 0 || summary.Totals.ElevationGainM < 0 {
		return fmt.Errorf("totals must not be negative")
	}
	for _, buckets := range [][]activities.SummaryBucket{summary.ByYear, summary.ByMonth, summary.BySport} {
		for _, bucket := range buckets {
			if bucket.ActivityCount < 0 || bucket.DistanceM < 0 || bucket.DistanceKnownCount < 0 || bucket.DistanceUnknownCount < 0 || bucket.DurationS < 0 || bucket.ElevationGainM < 0 {
				return fmt.Errorf("bucket %q must not be negative", bucket.Key)
			}
		}
	}
	return nil
}

func validateActivity(activity Activity) error {
	// A raw internal UUID here (rather than an "act_"-prefixed public ID)
	// would mean the sanitizer was bypassed for this record.
	if !strings.HasPrefix(activity.ID, ids.ActivityPrefix+"_") {
		return fmt.Errorf("id is not %s-prefixed", ids.ActivityPrefix)
	}
	if activity.DistanceM != nil && *activity.DistanceM < 0 {
		return fmt.Errorf("negative distance_m")
	}
	if activity.DurationS != nil && *activity.DurationS < 0 {
		return fmt.Errorf("negative duration_s")
	}
	if activity.MovingTimeS != nil && *activity.MovingTimeS < 0 {
		return fmt.Errorf("negative moving_time_s")
	}
	if activity.EndedAt != nil && activity.EndedAt.Before(activity.StartedAt) {
		return fmt.Errorf("ended_at before started_at")
	}
	return nil
}

func validateRouteFeature(feature RouteFeature) error {
	if feature.Type != "Feature" || feature.Geometry.Type != "LineString" {
		return fmt.Errorf("unexpected geometry type %q/%q", feature.Type, feature.Geometry.Type)
	}
	if !strings.HasPrefix(feature.Properties.ActivityID, ids.ActivityPrefix+"_") {
		return fmt.Errorf("activity_id is not %s-prefixed", ids.ActivityPrefix)
	}
	for _, point := range feature.Geometry.Coordinates {
		lon, lat := point[0], point[1]
		if lon < -180 || lon > 180 || lat < -90 || lat > 90 {
			return fmt.Errorf("coordinate (%f, %f) out of range", lon, lat)
		}
	}
	return nil
}
