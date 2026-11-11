package drifter

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
)

const snapshotWithCreatorsAndFinish = `{
  "version": "1.0",
  "generated_at": "2026-11-11T10:00:00Z",
  "source_account": "default",
  "has_more": false,
  "books": [
    {
      "source_provider": "weread",
      "source_book_id": "book-1",
      "title": "克拉拉与太阳",
      "author": "石黑一雄",
      "translator": "赵松",
      "publisher": "上海译文出版社",
      "isbn": "9787532786331",
      "cover_url": "https://example.test/klara.jpg",
      "reader_url": "https://weread.qq.com/web/bookDetail/book-1",
      "status": "completed",
      "progress_percent": 100,
      "reading_time_seconds": 5400,
      "last_read_at": "2024-03-04T00:20:00+08:00",
      "finished_at": "2024-03-04T00:30:00+08:00",
      "started_on": "2024-02-01",
      "is_custom": false,
      "matched_target": {
        "target_provider": "hardcover",
        "work_id": 9001,
        "edition_id": 42,
        "original_title": "Klara and the Sun",
        "original_author": "Kazuo Ishiguro"
      }
    }
  ]
}`

func TestParseSnapshotMapsCreatorsFinishAndRefs(t *testing.T) {
	// The reader finished the book at 00:30 in +08:00; UTC would put it on the
	// previous day, so the completion date must follow the import timezone.
	location := time.FixedZone("CST", 8*60*60)
	entries, err := ParseSnapshot([]byte(snapshotWithCreatorsAndFinish), location)
	if err != nil {
		t.Fatalf("ParseSnapshot() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ParseSnapshot() returned %d entries, want 1", len(entries))
	}
	media := entries[0]
	if media.Provider != sourceProviderWeRead || media.ExternalID != "book-1" || media.MediaType != "book" || media.ItemRole != "book" {
		t.Fatalf("identity mapping = %#v", media)
	}
	if media.Title != "克拉拉与太阳" || media.Language != "zh" || media.Publisher != "上海译文出版社" || media.CoverImageURL != "https://example.test/klara.jpg" {
		t.Fatalf("metadata mapping = %#v", media)
	}
	if media.Description != "" {
		t.Fatalf("Description = %q, want empty for a cataloged book", media.Description)
	}
	if len(media.Creators) != 2 {
		t.Fatalf("creators = %#v, want author and translator", media.Creators)
	}
	if media.Creators[0].Name != "石黑一雄" || media.Creators[0].Role != observations.MediaCreatorRoleAuthor || media.Creators[0].Provider != sourceProviderWeRead {
		t.Fatalf("author = %#v", media.Creators[0])
	}
	if media.Creators[1].Name != "赵松" || media.Creators[1].Role != observations.MediaCreatorRoleTranslator {
		t.Fatalf("translator = %#v", media.Creators[1])
	}
	if len(media.Titles) != 1 || !media.Titles[0].IsPrimary || media.Titles[0].Language != "zh" || media.Titles[0].TitleKind != "original" {
		t.Fatalf("titles = %#v", media.Titles)
	}

	refs := make(map[string]string, len(media.ExternalRefs))
	for _, ref := range media.ExternalRefs {
		refs[ref.Provider] = ref.ExternalID
	}
	if refs[sourceProviderWeRead] != "book-1" || refs[sourceProviderISBN] != "9787532786331" || refs[sourceProviderHardcover] != "42" {
		t.Fatalf("external refs = %#v", media.ExternalRefs)
	}

	if media.ProgressState == nil || media.ProgressState.Status != "completed" || media.ProgressState.ProgressPercent == nil || *media.ProgressState.ProgressPercent != 100 {
		t.Fatalf("progress state = %#v", media.ProgressState)
	}
	if media.Status != "completed" {
		t.Fatalf("status = %q, want completed", media.Status)
	}
	if media.ProgressState.LastUpdateAt == nil || !media.ProgressState.LastUpdateAt.Equal(time.Date(2024, time.March, 4, 0, 20, 0, 0, location)) {
		t.Fatalf("last update at = %v", media.ProgressState.LastUpdateAt)
	}
	if media.ProgressState.CompletedOn == nil || media.ProgressState.CompletedOn.String() != "2024-03-04" {
		t.Fatalf("completed on = %v, want 2024-03-04", media.ProgressState.CompletedOn)
	}
	if media.StartedOn == nil || media.StartedOn.String() != "2024-02-01" {
		t.Fatalf("started on = %v, want 2024-02-01", media.StartedOn)
	}

	if len(media.Events) != 1 {
		t.Fatalf("events = %#v, want one finished event", media.Events)
	}
	event := media.Events[0]
	if event.EventType != observations.MediaEventFinished || event.SourceEventID != "weread-finish-book-1" {
		t.Fatalf("finished event = %#v", event)
	}
	wantFinish := time.Date(2024, time.March, 3, 16, 30, 0, 0, time.UTC)
	if !event.EventAt.Equal(wantFinish) {
		t.Fatalf("finished event at = %v, want %v", event.EventAt, wantFinish)
	}
	if event.Note != "Read for 1h 30m" {
		t.Fatalf("finished event note = %q, want %q", event.Note, "Read for 1h 30m")
	}
}

func TestParseSnapshotMapsCustomBookAndHonestStatus(t *testing.T) {
	body := []byte(`{
  "version": "1.0",
  "has_more": false,
  "books": [
    {"source_book_id": "CB_custom_1", "title": "自印笔记", "status": "reading", "progress_percent": 42, "is_custom": true},
    {"source_book_id": "book-2", "title": "未开始的书", "author": "某作者", "status": "planned"},
    {"source_book_id": "book-3", "title": "弃读的书", "status": "abandoned"}
  ]
}`)
	entries, err := ParseSnapshot(body, time.UTC)
	if err != nil {
		t.Fatalf("ParseSnapshot() error = %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("ParseSnapshot() returned %d entries, want 3", len(entries))
	}
	if entries[0].Description != customUploadDescription {
		t.Fatalf("custom description = %q, want %q", entries[0].Description, customUploadDescription)
	}
	if entries[0].ProgressState.Status != "in_progress" || *entries[0].ProgressState.ProgressPercent != 42 {
		t.Fatalf("reading progress = %#v", entries[0].ProgressState)
	}
	if len(entries[0].Events) != 0 {
		t.Fatalf("custom book events = %#v, want none without a finish timestamp", entries[0].Events)
	}
	if entries[1].ProgressState.Status != "planned" {
		t.Fatalf("planned status = %q, want planned", entries[1].ProgressState.Status)
	}
	if entries[2].ProgressState.Status != "abandoned" {
		t.Fatalf("abandoned status = %q, want abandoned", entries[2].ProgressState.Status)
	}
}

func TestParseSnapshotRejectsBookWithoutIdentityOrTitle(t *testing.T) {
	for name, body := range map[string]string{
		"missing source_book_id": `{"books":[{"title":"无编号"}]}`,
		"missing title":          `{"books":[{"source_book_id":"book-4"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseSnapshot([]byte(body), time.UTC); err == nil {
				t.Fatal("ParseSnapshot() returned nil error, want an identity error")
			}
		})
	}
}
