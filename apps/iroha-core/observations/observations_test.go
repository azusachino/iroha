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
		Events:           []MediaEvent{{EventType: MediaEventWatched, EventAt: releaseDate, Position: &position, Rating: &rating}},
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

func TestPartialDatePreservesSourcePrecision(t *testing.T) {
	cases := []struct {
		name      string
		month     int
		day       int
		want      string
		precision DatePrecision
	}{
		{name: "year", want: "2024", precision: DatePrecisionYear},
		{name: "month", month: 3, want: "2024-03", precision: DatePrecisionMonth},
		{name: "day", month: 3, day: 9, want: "2024-03-09", precision: DatePrecisionDay},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			date, err := NewPartialDate(2024, test.month, test.day)
			if err != nil {
				t.Fatalf("NewPartialDate() error = %v", err)
			}
			if date.String() != test.want || date.Precision != test.precision {
				t.Fatalf("partial date = %q/%q, want %q/%q", date.String(), date.Precision, test.want, test.precision)
			}
		})
	}
}

func TestPartialDateRejectsInvalidProviderDay(t *testing.T) {
	if _, err := NewPartialDate(2024, 2, 31); err == nil {
		t.Fatal("NewPartialDate() accepted an impossible provider day")
	}
}
