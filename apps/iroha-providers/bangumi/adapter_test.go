package bangumi

import "testing"

func TestParseSnapshotMapsAnimeAndBookCollection(t *testing.T) {
	entries, err := ParseSnapshot([]byte(`{
  "total": 2, "data": [
    {"subject_type": 2, "type": 3, "rate": 8, "comment": "good", "tags": ["favorite"], "ep_status": 9, "updated_at": "2026-07-13T10:00:00+00:00", "subject": {"id": 20, "name": "NARUTO", "name_cn": "火影忍者", "date": "2002-10-03", "eps": 220, "images": {"large": "https://example.test/naruto.jpg"}}},
    {"subject_type": 1, "type": 1, "rate": 0, "vol_status": 2, "subject": {"id": 30, "name": "Book", "name_cn": "书", "platform": "novel", "volumes": 5}}
  ]
}`))
	if err != nil {
		t.Fatalf("ParseSnapshot() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ParseSnapshot() returned %d entries, want 2", len(entries))
	}
	if entries[0].MediaType != "anime_season" || entries[0].Progress == nil || *entries[0].Progress != 9 || len(entries[0].Titles) != 2 {
		t.Fatalf("anime mapping = %#v", entries[0])
	}
	if entries[0].CoverImageURL != "https://example.test/naruto.jpg" {
		t.Fatalf("CoverImageURL = %q, want the images.large URL", entries[0].CoverImageURL)
	}
	if entries[1].MediaType != "light_novel" || entries[1].ProgressState == nil || entries[1].ProgressState.Unit != "volumes" || entries[1].Score != nil {
		t.Fatalf("book mapping = %#v", entries[1])
	}
	if entries[1].CoverImageURL != "" {
		t.Fatalf("CoverImageURL = %q, want empty when images is absent", entries[1].CoverImageURL)
	}
	if len(entries[0].Events) != 0 {
		t.Fatalf("Bangumi snapshot events = %d, want no consumption events", len(entries[0].Events))
	}
	if entries[0].ProgressState.LastUpdateAt != nil {
		t.Fatalf("Bangumi last_update_at = %v, want nil because updated_at is unreliable provider metadata", entries[0].ProgressState.LastUpdateAt)
	}
	if entries[0].StateSourceID != "20" || entries[0].StateRatingScale == nil || *entries[0].StateRatingScale != 10 {
		t.Fatalf("Bangumi state provenance = %#v", entries[0])
	}
}
