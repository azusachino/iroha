package coverage

import (
	"testing"
	"time"
)

func TestCoverageIntervalsDeriveGaps(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(5 * 24 * time.Hour)
	covered := []Interval{
		{From: start.Add(2 * 24 * time.Hour), To: end},
		{From: start, To: start.Add(24 * time.Hour)},
	}

	gaps := complement(Interval{From: start, To: end}, mergeIntervals(covered))
	if len(gaps) != 1 || !gaps[0].From.Equal(start.Add(24*time.Hour)) || !gaps[0].To.Equal(start.Add(2*24*time.Hour)) {
		t.Fatalf("gaps = %#v, want the second day only", gaps)
	}
}

func TestSubtractPrefersNewerAssertionIntervals(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	input := Interval{From: start, To: start.Add(3 * 24 * time.Hour)}
	remaining := subtract(input, []Interval{{From: start.Add(24 * time.Hour), To: input.To}})
	if len(remaining) != 1 || !remaining[0].From.Equal(start) || !remaining[0].To.Equal(start.Add(24*time.Hour)) {
		t.Fatalf("remaining = %#v, want first day only", remaining)
	}
}
