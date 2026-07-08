package imports

import "testing"

func TestDecideSourceItem(t *testing.T) {
	same := "abc123"
	different := "def456"

	cases := []struct {
		name     string
		existing *string
		next     string
		want     sourceItemDecision
	}{
		{"no existing row is new", nil, "abc123", sourceItemNew},
		{"same hash is unchanged", &same, "abc123", sourceItemUnchanged},
		{"different hash is changed", &different, "abc123", sourceItemChanged},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decideSourceItem(tc.existing, tc.next)
			if got != tc.want {
				t.Errorf("decideSourceItem(%v, %q) = %v, want %v", tc.existing, tc.next, got, tc.want)
			}
		})
	}
}
