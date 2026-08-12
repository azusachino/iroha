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
