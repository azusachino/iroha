package observations

import (
	"testing"
	"time"
)

func TestMediaCarriesProviderNeutralOntology(t *testing.T) {
	releaseDate := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	season := 2
	episode := 9
	position := 9.0
	rating := 85.0
	media := Media{
		Provider:         "anilist",
		ExternalID:       "123",
		MediaType:        "anime_season",
		ItemRole:         "season",
		Title:            "Example Season 2",
		WorkExternalID:   "work-1",
		WorkKind:         "series",
		WorkTitle:        "Example",
		ParentExternalID: "season-1",
		ReleaseDate:      &releaseDate,
		SeasonNumber:     &season,
		EpisodeNumber:    &episode,
		Titles:           []MediaTitle{{Title: "Example Season 2", TitleKind: "primary", IsPrimary: true}},
		ExternalRefs:     []MediaExternalRef{{Provider: "mal", ExternalID: "456", MatchedBy: "provider_id"}},
		Events:           []MediaEvent{{EventType: "watched", Position: &position, Rating: &rating}},
		ProgressState:    &MediaProgress{Status: "in_progress", Position: &position},
	}

	if media.MediaType != "anime_season" || media.WorkExternalID == "" || media.ParentExternalID == "" {
		t.Fatalf("media item linkage was not retained: %#v", media)
	}
	if len(media.Titles) != 1 || len(media.ExternalRefs) != 1 || len(media.Events) != 1 {
		t.Fatalf("media graph slices = titles %d refs %d events %d, want 1 each", len(media.Titles), len(media.ExternalRefs), len(media.Events))
	}
	if media.ProgressState == nil || media.ProgressState.Position == nil || *media.ProgressState.Position != position {
		t.Fatalf("media progress = %#v, want position %.0f", media.ProgressState, position)
	}
}

func TestMediaSupportsBookFieldsAndNullableRatings(t *testing.T) {
	chapters := 42
	media := Media{
		Provider:     "bangumi",
		ExternalID:   "book-1",
		MediaType:    "manga",
		ItemRole:     "volume",
		Title:        "Example Volume",
		ChapterCount: &chapters,
		Score:        nil,
		Progress:     nil,
		ExternalRefs: []MediaExternalRef{{Provider: "bangumi", ExternalID: "book-1"}},
	}

	if media.MediaType != "manga" || media.ItemRole != "volume" || media.ChapterCount == nil || *media.ChapterCount != chapters {
		t.Fatalf("book fields were not retained: %#v", media)
	}
	if media.Score != nil || media.Progress != nil {
		t.Fatal("sparse provider ratings/progress must remain nullable")
	}
}
