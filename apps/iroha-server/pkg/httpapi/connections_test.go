package httpapi

import (
	"testing"
	"time"
)

func TestConnectionFreshnessDeadline(t *testing.T) {
	deadline := time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name      string
		now       time.Time
		delivered bool
		want      string
	}{
		{"fresh", deadline.Add(-time.Nanosecond), true, "within_cadence"},
		{"due", deadline, true, "overdue"},
		{"stale", deadline.Add(time.Hour), true, "overdue"},
		{"never delivered before due", deadline.Add(-time.Hour), false, "unknown"},
		{"never delivered due", deadline, false, "overdue"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := connectionFreshness(test.now, deadline, test.delivered); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}
