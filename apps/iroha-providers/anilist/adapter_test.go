package anilist

import "testing"

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
	if len(media.Events) != 3 {
		t.Fatalf("mapped events = %d, want list state + 2 rewatches", len(media.Events))
	}
	if media.Events[0].RatingScale == nil || *media.Events[0].RatingScale != 100 {
		t.Fatalf("list_state RatingScale = %v, want 100 (POINT_100 scoreFormat)", media.Events[0].RatingScale)
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
