package drifter

import (
	"bytes"
	"encoding/json"
	"os"
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

const snapshotWithReadingBook = `{
  "version": "1.0",
  "generated_at": "2026-11-12T02:15:30Z",
  "source_account": "default",
  "has_more": false,
  "books": [
    {
      "source_provider": "weread",
      "source_book_id": "book-9",
      "title": "福尔摩斯探案集",
      "author": "柯南·道尔",
      "status": "reading",
      "progress_percent": 12,
      "reading_time_seconds": 12337,
      "last_read_at": "2026-11-12T00:53:39+09:00",
      "started_on": "2026-11-01",
      "is_custom": false
    }
  ]
}`

func TestParseSnapshotEmitsDailyProgressEventForReadingBook(t *testing.T) {
	// 00:53+09:00 is still 2026-11-11 in UTC; the reading day must follow the
	// import timezone so the event lands on the day the reader experienced.
	location := time.FixedZone("JST", 9*60*60)
	entries, err := ParseSnapshot([]byte(snapshotWithReadingBook), location)
	if err != nil {
		t.Fatalf("ParseSnapshot() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ParseSnapshot() returned %d entries, want 1", len(entries))
	}
	media := entries[0]
	if media.Status != "in_progress" || media.ProgressState == nil || media.ProgressState.Status != "in_progress" {
		t.Fatalf("status mapping = %#v", media)
	}
	if len(media.Events) != 1 {
		t.Fatalf("events = %#v, want one progressed event", media.Events)
	}
	event := media.Events[0]
	if event.EventType != observations.MediaEventProgressed {
		t.Fatalf("event type = %q, want %q", event.EventType, observations.MediaEventProgressed)
	}
	if event.SourceEventID != "weread-progress-book-9:2026-11-12" {
		t.Fatalf("source event id = %q, want day-keyed weread-progress-book-9:2026-11-12", event.SourceEventID)
	}
	wantAt := time.Date(2026, time.November, 11, 15, 53, 39, 0, time.UTC)
	if !event.EventAt.Equal(wantAt) {
		t.Fatalf("event at = %v, want %v", event.EventAt, wantAt)
	}
	if event.ProgressPercent == nil || *event.ProgressPercent != 12 {
		t.Fatalf("progress percent = %#v, want 12", event.ProgressPercent)
	}
	if event.Note != "" {
		t.Fatalf("note = %q, want empty: cumulative reading time must not be stamped per day", event.Note)
	}
}

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

func TestParseSnapshotMapsCustomBookAndFiltersUnreadPlanned(t *testing.T) {
	body := []byte(`{
  "version": "1.0",
  "has_more": false,
  "books": [
    {"source_book_id": "CB_custom_1", "title": "自印笔记", "status": "reading", "progress_percent": 42, "is_custom": true},
    {"source_book_id": "book-2", "title": "未开始的书", "author": "某作者", "status": "planned"},
    {"source_book_id": "book-3", "title": "弃读的书", "status": "abandoned"},
    {"source_book_id": "book-4", "title": "已读的书", "author": "某作者", "status": "completed", "progress_percent": 100}
  ]
}`)
	entries, err := ParseSnapshot(body, time.UTC)
	if err != nil {
		t.Fatalf("ParseSnapshot() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("ParseSnapshot() returned %d entries, want 2 (only active and completed books)", len(entries))
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
	if entries[1].ExternalID != "book-4" || entries[1].ProgressState.Status != "completed" {
		t.Fatalf("completed book = %#v", entries[1])
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

// goldenPagePath is a copy of drifter's published golden page; see
// testdata/README.md for the drifter commit it came from.
const goldenPagePath = "testdata/reading-snapshot-v1.golden.json"

func TestParseSnapshotReadsDrifterGoldenPage(t *testing.T) {
	body, err := os.ReadFile(goldenPagePath)
	if err != nil {
		t.Fatalf("read golden page: %v", err)
	}
	// Every field drifter publishes must have a home in the wire structs.
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var page ReadingSnapshot
	if err := decoder.Decode(&page); err != nil {
		t.Fatalf("golden page has fields the wire structs do not read: %v", err)
	}
	if page.Watermark == "" || !page.HasMore || page.Cursor == "" {
		t.Fatalf("golden envelope = watermark %q has_more %v cursor %q", page.Watermark, page.HasMore, page.Cursor)
	}
	entries, err := ParseSnapshot(body, time.UTC)
	if err != nil {
		t.Fatalf("ParseSnapshot(golden) error = %v", err)
	}
	if len(entries) != 1 || entries[0].Title != "克拉拉与太阳" {
		t.Fatalf("golden entries = %#v, want the one completed book", entries)
	}
}

func TestParseSnapshotRefusesUnknownMajorVersion(t *testing.T) {
	for _, version := range []string{"2.0", ""} {
		body := []byte(`{"version":"` + version + `","has_more":false,"books":[]}`)
		if _, err := ParseSnapshot(body, time.UTC); err == nil {
			t.Fatalf("ParseSnapshot(version %q) returned nil error", version)
		}
	}
	if _, err := ParseSnapshot([]byte(`{"version":"1.3","has_more":false,"books":[]}`), time.UTC); err != nil {
		t.Fatalf("ParseSnapshot(minor bump 1.3) error = %v, want accepted", err)
	}
}

// A mapping is the source's own confirmed statement, so its ref is marked as a
// source mapping and the adapter declares the provider it owns. A book listed
// without a matched target carries no such ref, which is what lets a full
// snapshot withdraw the one stored earlier.
func TestParseSnapshotMarksMappingRefAndDeclaresItsProvider(t *testing.T) {
	mapped, err := ParseSnapshot([]byte(`{"version":"1.0","books":[{"source_book_id":"b1","title":"T","status":"reading","matched_target":{"target_provider":"hardcover","work_id":7,"edition_id":42}}]}`), time.UTC)
	if err != nil || len(mapped) != 1 {
		t.Fatalf("mapped page = %v, %v", mapped, err)
	}
	var found bool
	for _, ref := range mapped[0].ExternalRefs {
		if ref.Provider == sourceProviderHardcover {
			found = ref.MatchedBy == observations.MediaMatchSourceMapping
		}
	}
	if !found {
		t.Fatalf("hardcover ref = %#v, want matched_by %q", mapped[0].ExternalRefs, observations.MediaMatchSourceMapping)
	}
	unmapped, err := ParseSnapshot([]byte(`{"version":"1.0","books":[{"source_book_id":"b1","title":"T","status":"reading","matched_target":null}]}`), time.UTC)
	if err != nil || len(unmapped) != 1 {
		t.Fatalf("unmapped page = %v, %v", unmapped, err)
	}
	if unmapped[0].MappedRefProvider != sourceProviderHardcover {
		t.Fatalf("MappedRefProvider = %q, want %q", unmapped[0].MappedRefProvider, sourceProviderHardcover)
	}
	for _, ref := range unmapped[0].ExternalRefs {
		if ref.Provider == sourceProviderHardcover {
			t.Fatalf("unmapped book carries a hardcover ref: %#v", ref)
		}
	}
}
