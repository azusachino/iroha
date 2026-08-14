package metrics

import (
	"errors"
	"testing"
	"time"
)

func TestBuildPeriodsProducesCompleteHalfOpenRange(t *testing.T) {
	location, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	periods, err := BuildPeriods(
		time.Date(2026, time.January, 1, 0, 0, 0, 0, location),
		time.Date(2026, time.April, 1, 0, 0, 0, 0, location),
		"month",
		location,
	)
	if err != nil {
		t.Fatalf("build periods: %v", err)
	}
	want := []string{"2026-01", "2026-02", "2026-03"}
	if len(periods) != len(want) {
		t.Fatalf("periods = %v, want %v", periods, want)
	}
	for index := range want {
		if periods[index] != want[index] {
			t.Fatalf("periods = %v, want %v", periods, want)
		}
	}
}

func TestBuildPeriodsRejectsInvalidGrainAndBoundary(t *testing.T) {
	location := time.UTC
	from := time.Date(2026, time.January, 2, 0, 0, 0, 0, location)
	to := time.Date(2026, time.February, 1, 0, 0, 0, 0, location)
	if _, err := BuildPeriods(from, to, "month", location); !errors.Is(err, ErrInvalidPeriod) {
		t.Fatalf("boundary error = %v, want invalid period", err)
	}
	if _, err := BuildPeriods(from, to, "week", location); !errors.Is(err, ErrUnsupportedGrain) {
		t.Fatalf("grain error = %v, want unsupported grain", err)
	}
}
