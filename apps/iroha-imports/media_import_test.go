package imports

import (
	"testing"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
)

func TestTitleCandidates(t *testing.T) {
	confidence := 0.75
	titles := []observations.MediaTitle{
		{Title: "", TitleKind: "original", Provider: "bangumi"},
		{
			Title:      "ノルウェイの森",
			Language:   "ja",
			Script:     "jpan",
			Region:     "JP",
			TitleKind:  "original",
			Provider:   "anilist",
			IsPrimary:  true,
			Confidence: &confidence,
		},
	}

	candidates := titleCandidates(titles)
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1 (empty titles are dropped)", len(candidates))
	}
	got := candidates[0]
	if got.Title != "ノルウェイの森" || got.Language != "ja" || got.Script != "jpan" ||
		got.Region != "JP" || got.TitleKind != "original" || got.Provider != "anilist" || !got.IsPrimary {
		t.Errorf("candidate = %+v, want every claim field carried across", got)
	}
	if got.Confidence == nil || *got.Confidence != confidence {
		t.Errorf("Confidence = %v, want %v", got.Confidence, confidence)
	}
}
