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

func TestImportDisposition(t *testing.T) {
	cases := []struct {
		name             string
		priorSameVersion bool
		priorAnyVersion  bool
		want             importDisposition
	}{
		{"no prior completed import at all is fresh", false, false, dispositionFresh},
		{"prior completed import at same parser_version is skip", true, true, dispositionSkip},
		{"prior completed import at a different parser_version is reprocess", false, true, dispositionReprocess},
		// priorSameVersion implies priorAnyVersion, but even if a caller
		// passes an inconsistent combination, same-version must win: an
		// exact-match prior import always means skip, never reprocess.
		{"same-version prior takes precedence over any-version flag", true, false, dispositionSkip},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decideImportDisposition(tc.priorSameVersion, tc.priorAnyVersion)
			if got != tc.want {
				t.Errorf("decideImportDisposition(%v, %v) = %v, want %v", tc.priorSameVersion, tc.priorAnyVersion, got, tc.want)
			}
		})
	}
}
