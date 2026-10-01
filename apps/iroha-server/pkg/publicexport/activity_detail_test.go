package publicexport

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
)

func TestValidateActivityDetails_OK(t *testing.T) {
	id := "act_019f82a5-87b2-7b31-9ebf-19f169899a76"
	distance := 100.0
	duration := 60
	details := map[string]ActivityDetail{
		id: {
			Activity: ActivityDetailActivity{
				Activity: Activity{ID: id, StartedAt: time.Unix(0, 0), DistanceM: &distance, DurationS: &duration},
			},
			Route: []ActivityDetailRoutePoint{{Lat: 35, Lon: 139}},
		},
	}
	if err := ValidateActivityDetails(details); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestToActivityDetailLapsFiltersPlaceholderRows(t *testing.T) {
	distance := 550.0
	duration := 210
	laps := []models.ActivityLap{
		{LapNo: 1},
		{LapNo: 2, DistanceM: &distance, DurationS: &duration},
	}

	got := toActivityDetailLaps(laps)
	if len(got) != 1 || got[0].LapNo != 2 {
		t.Fatalf("expected only the complete lap, got %#v", got)
	}
}

func TestDecimateActivityDetailRouteKeepsEndpoints(t *testing.T) {
	points := make([]ActivityDetailRoutePoint, activityDetailMaxRoutePoints+50)
	for i := range points {
		points[i].Seq = i
	}

	got := decimateActivityDetailRoute(points)
	if len(got) > activityDetailMaxRoutePoints+1 {
		t.Fatalf("expected at most %d points, got %d", activityDetailMaxRoutePoints+1, len(got))
	}
	if got[0].Seq != 0 || got[len(got)-1].Seq != len(points)-1 {
		t.Fatalf("expected endpoints to survive, got first=%d last=%d", got[0].Seq, got[len(got)-1].Seq)
	}
}

func TestToActivityDetailRoute_TrimmingAndRounding(t *testing.T) {
	// Construct a 1 km track (~100 points, ~10m apart) with high precision coordinates
	const metersPerDegLat = 111000.0
	points := make([]models.ActivityRoutePoint, 100)
	for i := range points {
		points[i] = models.ActivityRoutePoint{
			Seq: i,
			Lon: 139.712345678,
			Lat: 35.000000001 + float64(i)*10.0/metersPerDegLat,
		}
	}

	got := toActivityDetailRoute(points, true)
	if len(got) == 0 {
		t.Fatal("expected non-empty trimmed route")
	}

	// Coordinates should be rounded to 5 decimal places
	if got[0].Lon != 139.71235 {
		t.Errorf("got lon = %f, want 139.71235", got[0].Lon)
	}

	// Endpoints must have been trimmed by ~200m
	if got[0].Seq < 15 {
		t.Errorf("start point seq = %d, expected >= 15 after 200m trim", got[0].Seq)
	}
	if got[len(got)-1].Seq > 85 {
		t.Errorf("end point seq = %d, expected <= 85 after 200m trim", got[len(got)-1].Seq)
	}

	// Short track under 400m should be dropped
	shortPoints := points[:30] // ~300m
	gotShort := toActivityDetailRoute(shortPoints, true)
	if len(gotShort) != 0 {
		t.Errorf("expected short route to be dropped, got %d points", len(gotShort))
	}
}

func TestValidateActivityDetails_AcceptsAnyPublicActivityID(t *testing.T) {
	id := "act_0198f8f0-0000-7000-8000-000000000000"
	if err := ValidateActivityDetails(map[string]ActivityDetail{
		id: {Activity: ActivityDetailActivity{Activity: Activity{ID: id, StartedAt: time.Unix(0, 0)}}},
	}); err != nil {
		t.Fatalf("expected any public activity to pass, got %v", err)
	}
}

func TestValidateActivityDetails_RejectsOutOfRangeRoute(t *testing.T) {
	id := "act_019f82a5-87b2-7b31-9ebf-19f169899a76"
	if err := ValidateActivityDetails(map[string]ActivityDetail{
		id: {
			Activity: ActivityDetailActivity{Activity: Activity{ID: id, StartedAt: time.Unix(0, 0)}},
			Route:    []ActivityDetailRoutePoint{{Lat: 91, Lon: 139}},
		},
	}); err == nil {
		t.Fatal("expected invalid route point to fail")
	}
}
