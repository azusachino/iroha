package publicexport

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/google/uuid"
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

func TestToActivityPreservesDerivedMetrics(t *testing.T) {
	distance := 1234.5
	duration := 3600
	moving := 3300
	elevation := 42.5
	avgHR := 145
	maxHR := 178
	pace := 330.0

	got := ToActivity(models.Activity{
		ID:             uuid.New(),
		SportType:      "run",
		StartedAt:      time.Unix(0, 0),
		DistanceM:      &distance,
		DurationS:      &duration,
		MovingTimeS:    &moving,
		ElevationGainM: &elevation,
		AvgHR:          &avgHR,
		MaxHR:          &maxHR,
		AvgPaceSPerKM:  &pace,
	})

	if got.MovingTimeS == nil || *got.MovingTimeS != moving {
		t.Fatalf("moving time = %v, want %d", got.MovingTimeS, moving)
	}
	if got.ElevationGainM == nil || *got.ElevationGainM != elevation {
		t.Fatalf("elevation gain = %v, want %.1f", got.ElevationGainM, elevation)
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
