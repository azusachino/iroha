package anilist

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

func TestParseSnapshotMapsAnimeEntryToMediaGraph(t *testing.T) {
	entries, err := ParseSnapshot([]byte(`{
  "data": {"User": {"mediaListOptions": {"scoreFormat": "POINT_100"}}, "MediaListCollection": {"lists": [{"entries": [{
    "id": 7, "status": "REPEATING", "score": 85, "progress": 9, "repeat": 2,
    "notes": "great season", "startedAt": {"year": 2024, "month": 1, "day": 2},
    "completedAt": {"year": 2024, "month": 3, "day": 4}, "updatedAt": 1709500000,
    "media": {
      "id": 123, "idMal": 456, "type": "ANIME", "format": "TV", "episodes": 12,
      "title": {"romaji": "Example", "english": "Example Show", "native": "例示"},
      "coverImage": {"large": "https://example.test/cover.jpg"},
      "description": "  A show about examples.  ",
      "relations": {"edges": [{"relationType": "SEQUEL", "node": {"id": 124}}]}
    }
  }]}]}}
}`))
	if err != nil {
		t.Fatalf("ParseSnapshot() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ParseSnapshot() returned %d entries, want 1", len(entries))
	}
	media := entries[0]
	if media.MediaType != "anime_season" || media.ItemRole != "season" || media.Title != "Example Show" {
		t.Fatalf("mapped identity = %#v", media)
	}
	if media.CoverImageURL != "https://example.test/cover.jpg" {
		t.Fatalf("CoverImageURL = %q, want the coverImage.large URL", media.CoverImageURL)
	}
	if media.Description != "A show about examples." {
		t.Fatalf("Description = %q, want the trimmed description text", media.Description)
	}
	if len(media.Titles) != 3 || len(media.ExternalRefs) != 2 || len(media.Relations) != 1 {
		t.Fatalf("mapped graph sizes = titles %d refs %d relations %d, want 3/2/1", len(media.Titles), len(media.ExternalRefs), len(media.Relations))
	}
	if media.ProgressState == nil || media.ProgressState.Paused || media.ProgressState.PlayCount != 2 {
		t.Fatalf("mapped progress = %#v", media.ProgressState)
	}
	if len(media.Events) != 0 {
		t.Fatalf("mapped events = %d, want no consumption events for a library snapshot", len(media.Events))
	}
	if media.StateSourceID != "7" || media.StateRatingScale == nil || *media.StateRatingScale != 100 {
		t.Fatalf("state provenance = id %q rating scale %v, want entry 7 and 100", media.StateSourceID, media.StateRatingScale)
	}
	if media.StartedOn == nil || media.StartedOn.String() != "2024-01-02" || media.CompletedOn == nil || media.CompletedOn.String() != "2024-03-04" {
		t.Fatalf("partial dates = %v/%v", media.StartedOn, media.CompletedOn)
	}
}

func TestParseSnapshotMapsMangaAndNullableScore(t *testing.T) {
	entries, err := ParseSnapshot([]byte(`{
  "data": {"MediaListCollection": {"lists": [{"entries": [{
    "id": 8, "status": "PLANNING", "score": 0, "progressVolumes": 2,
    "media": {"id": 321, "type": "MANGA", "format": "MANGA", "chapters": 42,
      "title": {"native": "例示漫画"}}
  }]}]}}
}`))
	if err != nil {
		t.Fatalf("ParseSnapshot() error = %v", err)
	}
	media := entries[0]
	if media.MediaType != "manga" || media.ItemRole != "series" || media.Progress == nil || *media.Progress != 2 || media.Score != nil {
		t.Fatalf("mapped manga fields = %#v", media)
	}
	if media.ProgressState == nil || media.ProgressState.Unit != "volumes" {
		t.Fatalf("mapped manga progress = %#v", media.ProgressState)
	}
}

func TestFuzzyDateDoesNotBecomeAnInstant(t *testing.T) {
	date := anilistDate{Year: 2024, Month: 3}
	partial := date.Partial()
	if partial == nil || partial.String() != "2024-03" {
		t.Fatalf("partial date = %#v, want 2024-03", partial)
	}
	if date.Time() != nil {
		t.Fatal("year-month provider date became an exact timestamp")
	}

	invalid := anilistDate{Year: 2024, Day: 31}
	if invalid.Partial() != nil || invalid.Time() != nil {
		t.Fatal("provider day without a month was accepted")
	}
}

func TestImportMediaHistoryEmitsConsumptionEvents(t *testing.T) {
	entries, err := NewAdapter().ImportMediaHistory(context.Background(), provider.Source{
		Kind: ActivitySourceKind,
		Open: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(`{
  "data": {"Page": {"activities": [
    {"id": 21, "status": "completed", "progress": "12", "createdAt": 1786752000,
     "media": {"id": 111, "type": "ANIME", "format": "TV", "episodes": 12,
       "title": {"romaji": "Kanata no Astra"}}},
    {"id": 22, "status": "progressed", "progress": "chapter 5 - 8", "createdAt": 1786838400,
     "media": {"id": 321, "type": "MANGA", "format": "MANGA", "chapters": 42,
       "title": {"native": "例示漫画"}}},
    {"id": 23, "status": "paused", "progress": "3", "createdAt": 1786924800,
     "media": {"id": 111, "type": "ANIME", "format": "TV", "episodes": 12,
       "title": {"romaji": "Kanata no Astra"}}},
    {"id": 24, "status": "progressed", "progress": "", "createdAt": 1787011200,
     "media": {"id": 111, "type": "ANIME", "format": "TV", "episodes": 12,
       "title": {"romaji": "Kanata no Astra"}}}
  ]}}
}`)), nil
		},
	}, provider.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportMediaHistory() error = %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("history entries = %d, want 4", len(entries))
	}
	completed := entries[0]
	if len(completed.Media.Events) != 1 {
		t.Fatalf("completed anime events = %#v, want one", completed.Media.Events)
	}
	event := completed.Media.Events[0]
	if event.EventType != observations.MediaEventCompleted || event.SourceEventID != "activity:21" {
		t.Fatalf("completed event = %#v", event)
	}
	if !event.EventAt.Equal(time.Unix(1786752000, 0).UTC()) || event.Unit != "episodes" || event.Total == nil || *event.Total != 12 {
		t.Fatalf("completed event detail = %#v", event)
	}
	progressed := entries[1]
	if len(progressed.Media.Events) != 1 {
		t.Fatalf("progressed manga events = %#v, want one", progressed.Media.Events)
	}
	event = progressed.Media.Events[0]
	if event.EventType != observations.MediaEventProgressed || event.SourceEventID != "activity:22" {
		t.Fatalf("progressed event = %#v", event)
	}
	if event.Unit != "chapters" || event.Position == nil || *event.Position != 8 || event.Total == nil || *event.Total != 42 {
		t.Fatalf("progressed event keeps range end and total = %#v", event)
	}
	if event.Note != "" {
		t.Fatalf("note = %q, want empty", event.Note)
	}
	for _, entry := range entries[2:] {
		if len(entry.Media.Events) != 0 {
			t.Fatalf("state-only activity emitted events = %#v", entry.Media.Events)
		}
	}
}

func TestImportMediaHistoryMapsDatedMangaProgress(t *testing.T) {
	entries, err := NewAdapter().ImportMediaHistory(context.Background(), provider.Source{
		Kind: ActivitySourceKind,
		Open: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader(`{
  "data": {"Page": {"activities": [{
    "id": 9, "status": "read", "progress": "chapter 2", "createdAt": 1786752000,
    "media": {"id": 321, "type": "MANGA", "format": "MANGA", "chapters": 42,
      "title": {"native": "例示漫画"}}
  }]}}
}`)), nil
		},
	}, provider.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportMediaHistory() error = %v", err)
	}
	if len(entries) != 1 || len(entries[0].Updates) != 1 {
		t.Fatalf("history = %#v", entries)
	}
	update := entries[0].Updates[0]
	if update.SourceEventID != "activity:9" || update.Status != "in_progress" || update.Unit != "chapters" || update.Position == nil || *update.Position != 2 {
		t.Fatalf("update = %#v", update)
	}
	if update.EffectiveAt != time.Unix(1786752000, 0).UTC() || update.Note != "read chapter 2" {
		t.Fatalf("timing/note = %v/%q", update.EffectiveAt, update.Note)
	}
}
