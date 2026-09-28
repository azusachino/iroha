package publicexport

import (
	"context"
	"fmt"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
)

// Snapshot is the sanitized public projection served live by /public/v1: the
// same summary, activity list, routes, and meta a static export writes.
type Snapshot struct {
	Summary    activities.Summary
	Activities []Activity
	Routes     RouteFeatureCollection
	Meta       Meta
}

// BuildSnapshot computes and validates the public projection. It never
// returns data that failed the privacy/schema invariants in Validate.
func BuildSnapshot(ctx context.Context, activitySvc *activities.Service, geocodeSvc *geocode.Service, now time.Time) (Snapshot, error) {
	summary, err := Summary(activitySvc, "", "")
	if err != nil {
		return Snapshot{}, fmt.Errorf("build summary: %w", err)
	}
	list, err := collectAllActivities(activitySvc)
	if err != nil {
		return Snapshot{}, fmt.Errorf("collect activities: %w", err)
	}
	routes, err := Routes(ctx, activitySvc, geocodeSvc, false)
	if err != nil {
		return Snapshot{}, fmt.Errorf("build routes: %w", err)
	}
	if err := Validate(summary, list, routes); err != nil {
		return Snapshot{}, fmt.Errorf("validate public projection: %w", err)
	}
	return Snapshot{
		Summary:    summary,
		Activities: list,
		Routes:     routes,
		Meta:       Meta{GeneratedAt: now.UTC(), RoutesIncluded: true, ActivityCount: len(list)},
	}, nil
}

// Detail returns the validated public detail of one activity.
func Detail(activitySvc *activities.Service, id string) (ActivityDetail, bool, error) {
	_, found, err := activitySvc.Get(id)
	if err != nil || !found {
		return ActivityDetail{}, false, err
	}
	details, err := ActivityDetails(activitySvc, []Activity{{ID: id}}, true)
	if err != nil {
		return ActivityDetail{}, false, err
	}
	if err := ValidateActivityDetails(details); err != nil {
		return ActivityDetail{}, false, fmt.Errorf("validate public detail: %w", err)
	}
	return details[id], true, nil
}
