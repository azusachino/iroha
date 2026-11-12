package httpapi

import (
	"testing"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
)

func TestDuplicatesEvent(t *testing.T) {
	position := 3.0
	percent := 45.0
	cases := []struct {
		name   string
		change media.Change
		want   bool
	}{
		{"progress position duplicates progressed event", media.Change{Position: &position}, true},
		{"progress percent duplicates progressed event", media.Change{ProgressPercent: &percent}, true},
		{"completed status duplicates completed event", media.Change{Status: "completed"}, true},
		{"in progress status duplicates event", media.Change{Status: "in_progress"}, true},
		{"planned status has no event counterpart", media.Change{Status: "planned"}, false},
		{"paused status has no event counterpart", media.Change{Status: "paused"}, false},
		{"abandoned status has no event counterpart", media.Change{Status: "abandoned"}, false},
		{"rating-only change stays visible", media.Change{Status: "planned", Rating: ptr(8.5)}, false},
		{"empty change stays visible", media.Change{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := duplicatesEvent(tc.change); got != tc.want {
				t.Fatalf("duplicatesEvent(%+v) = %v, want %v", tc.change, got, tc.want)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
