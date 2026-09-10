package imports

import "testing"

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
		// An exact-match prior import must never be reprocessed.
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
