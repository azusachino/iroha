package daily

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundTrip(t *testing.T) {
	want := Cursor{
		Day: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		ID:  uuid.MustParse("018cc251-7b2e-7d52-9b0d-6bd6f2c9c9e4"),
	}

	got, err := DecodeCursor(EncodeCursor(want))
	if err != nil {
		t.Fatalf("DecodeCursor returned error: %v", err)
	}
	if !got.Day.Equal(want.Day) || got.ID != want.ID {
		t.Fatalf("cursor = %+v, want %+v", got, want)
	}
}

func TestDecodeCursorRejectsMalformedValue(t *testing.T) {
	for _, value := range []string{"", "not-base64", "MjAyNC0wMS0wMg==", "MjAyNC0wMS0wMnxibnVsbA"} {
		if _, err := DecodeCursor(value); err == nil {
			t.Errorf("DecodeCursor(%q) returned nil error", value)
		}
	}
}

func TestMergeAggregateRowsCombinesRingMetricAndDayBuckets(t *testing.T) {
	jan := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	got := mergeAggregateRows(
		[]ringAggregateRow{{Period: jan, MoveAvg: 500, ExerciseAvg: 30, StandAvg: 10, RingDays: 2, MoveClosed: 1}},
		[]metricAggregateRow{{Period: jan, Metric: "steps", Avg: 9000}, {Period: feb, Metric: "resting_hr", Avg: 54}},
		[]dayAggregateRow{{Period: jan, Days: 2}, {Period: feb, Days: 1}},
	)

	if len(got) != 2 || !got[0].Period.Equal(jan) || !got[1].Period.Equal(feb) {
		t.Fatalf("buckets = %#v, want chronological January and February buckets", got)
	}
	if got[0].Days != 2 || got[0].MoveKcalAvg != 500 || got[0].MoveClosedPct != 50 || got[0].Metrics["steps"] != 9000 {
		t.Fatalf("January bucket = %#v", got[0])
	}
	if got[1].Days != 1 || got[1].Metrics["resting_hr"] != 54 {
		t.Fatalf("February bucket = %#v", got[1])
	}
}
