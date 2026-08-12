package publicexport

import (
	"testing"
	"time"
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

func TestValidateActivityDetails_RejectsUnapprovedID(t *testing.T) {
	id := "act_0198f8f0-0000-7000-8000-000000000000"
	if err := ValidateActivityDetails(map[string]ActivityDetail{
		id: {Activity: ActivityDetailActivity{Activity: Activity{ID: id, StartedAt: time.Unix(0, 0)}}},
	}); err == nil {
		t.Fatal("expected unapproved activity to fail")
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
