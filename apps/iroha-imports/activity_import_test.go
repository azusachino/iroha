package imports

import (
	"testing"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
)

func TestNormalizeTitleOrigin(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "taxonomy label", input: observations.TitleOriginSystem, want: observations.TitleOriginSystem},
		{name: "user title", input: observations.TitleOriginCustom, want: observations.TitleOriginCustom},
		{name: "explicit unknown", input: observations.TitleOriginUnknown, want: observations.TitleOriginUnknown},
		{name: "empty stays inside the constraint", input: "", want: observations.TitleOriginUnknown},
		{name: "unrecognized value does not violate the check constraint", input: "guessed", want: observations.TitleOriginUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeTitleOrigin(tt.input); got != tt.want {
				t.Errorf("normalizeTitleOrigin(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
