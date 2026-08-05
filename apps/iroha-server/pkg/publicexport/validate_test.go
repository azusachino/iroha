package publicexport

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
)

func validSummary() activities.Summary {
	return activities.Summary{
		Totals: activities.SummaryTotals{ActivityCount: 1, DistanceM: 100, DurationS: 60, MovingTimeS: 60},
	}
}

func validActivity() Activity {
	return Activity{ID: "act_" + "0198f8f0-0000-7000-8000-000000000000", StartedAt: time.Unix(0, 0)}
}

func TestValidate_OK(t *testing.T) {
	if err := Validate(validSummary(), []Activity{validActivity()}, RouteFeatureCollection{}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidate_RejectsUnprefixedActivityID(t *testing.T) {
	activity := validActivity()
	activity.ID = "0198f8f0-0000-7000-8000-000000000000"
	if err := Validate(validSummary(), []Activity{activity}, RouteFeatureCollection{}); err == nil {
		t.Fatal("expected error for unprefixed activity id")
	}
}

func TestValidate_RejectsNegativeTotals(t *testing.T) {
	summary := validSummary()
	summary.Totals.DistanceM = -1
	if err := Validate(summary, nil, RouteFeatureCollection{}); err == nil {
		t.Fatal("expected error for negative totals")
	}
}

func TestValidate_RejectsEndedBeforeStarted(t *testing.T) {
	activity := validActivity()
	before := activity.StartedAt.Add(-time.Hour)
	activity.EndedAt = &before
	if err := Validate(validSummary(), []Activity{activity}, RouteFeatureCollection{}); err == nil {
		t.Fatal("expected error for ended_at before started_at")
	}
}

func TestValidate_RejectsOutOfRangeCoordinate(t *testing.T) {
	routes := RouteFeatureCollection{Features: []RouteFeature{{
		Type:     "Feature",
		Geometry: RouteGeometry{Type: "LineString", Coordinates: [][2]float64{{200, 0}}},
	}}}
	if err := Validate(validSummary(), []Activity{validActivity()}, routes); err == nil {
		t.Fatal("expected error for out-of-range coordinate")
	}
}
