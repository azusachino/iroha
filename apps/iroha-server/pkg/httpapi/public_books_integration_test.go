//go:build integration

package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func seedBookAuthor(t *testing.T, db *gorm.DB, itemID uuid.UUID, name string) {
	t.Helper()
	now := time.Now().UTC()
	creator := uuid.New()
	if err := db.Exec(`insert into tb_media_creators (id, name, sort_name, created_at, updated_at)
		values (?, ?, ?, ?, ?)`, creator, name, name, now, now).Error; err != nil {
		t.Fatalf("seed creator: %v", err)
	}
	if err := db.Exec(`insert into tb_media_creator_roles (id, creator_id, scope_type, scope_id, role, provider, created_at)
		values (?, ?, 'item', ?, 'author', 'test', ?)`, uuid.New(), creator, itemID, now).Error; err != nil {
		t.Fatalf("seed creator role: %v", err)
	}
}

func seedBookProgress(t *testing.T, db *gorm.DB, itemID uuid.UUID, status string, completedOn *time.Time, lastUpdateAt *time.Time) {
	t.Helper()
	precision := ""
	if completedOn != nil {
		precision = "day"
	}
	if err := db.Exec(`insert into tb_media_progress (media_item_id, status, completed_on_value, completed_on_precision, last_update_at, updated_at)
		values (?, ?, ?, ?, ?, ?)`, itemID, status, completedOn, precision, lastUpdateAt, time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed book progress: %v", err)
	}
}

func bumpMediaRevision(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`insert into tb_read_revisions (namespace, revision, updated_at)
		values ('read_media', 1, now())
		on conflict (namespace) do update set revision = tb_read_revisions.revision + 1, updated_at = now()`).Error; err != nil {
		t.Fatalf("bump media revision: %v", err)
	}
}

// TestIntegrationPublicBooksAreSanitizedAndRevisionFresh drives the books part
// of the public projection: only books appear, only allowlisted fields ship,
// and a media revision change rebuilds the cached snapshot.
func TestIntegrationPublicBooksAreSanitizedAndRevisionFresh(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	resetMediaTables(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })
	server := newIntegrationServer(t, db)

	finished := time.Date(2024, 3, 4, 0, 30, 0, 0, time.UTC)
	recent := time.Date(2026, 11, 12, 1, 22, 14, 0, time.UTC)

	bookWork := seedWork(t, db, "book")
	finishedBook := seedItem(t, db, bookWork, "book", "克拉拉与太阳")
	seedBookAuthor(t, db, finishedBook, "石黑一雄")
	seedBookProgress(t, db, finishedBook, "completed", &finished, &finished)

	readingWork := seedWork(t, db, "book")
	readingBook := seedItem(t, db, readingWork, "book", "如何了解一个人")
	seedBookProgress(t, db, readingBook, "in_progress", nil, &recent)

	animeWork := seedWork(t, db, "anime")
	anime := seedItem(t, db, animeWork, "anime", "Dragon Maid")
	seedBookProgress(t, db, anime, "completed", &finished, &finished)

	// A book-shaped private source id must never reach the public payload.
	if err := db.Exec(`insert into tb_media_external_refs
		(id, scope_type, scope_id, provider, external_id, matched_by, confidence, created_at)
		values (?, 'item', ?, 'weread', '41077357', 'source_identity', 1.0, ?)`,
		uuid.New(), finishedBook, time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed external ref: %v", err)
	}

	bumpMediaRevision(t, db)
	rec := publicGet(t, server, "/public/v1/books")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET books = %d %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=86400" {
		t.Fatalf("Cache-Control = %q", got)
	}
	var books []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &books); err != nil {
		t.Fatalf("decode books: %v", err)
	}
	if len(books) != 2 {
		t.Fatalf("books = %d, want 2 (reading book first)", len(books))
	}
	first := books[0]
	if first["title"] != "如何了解一个人" || first["status"] != "in_progress" {
		t.Fatalf("first book = %v, want most recently touched", first)
	}
	id, _ := first["id"].(string)
	if len(id) < 6 || id[:6] != "media_" {
		t.Fatalf("book id %q is not media_-prefixed", id)
	}
	if _, ok := first["finished_at"]; ok {
		t.Fatalf("in-progress book ships finished_at: %v", first)
	}
	if _, ok := first["last_read_at"]; ok {
		t.Fatalf("progress update was incorrectly published as a reading time: %v", first)
	}
	second := books[1]
	if second["title"] != "克拉拉与太阳" || second["status"] != "completed" {
		t.Fatalf("second book = %v", second)
	}
	authors, _ := second["authors"].([]any)
	if len(authors) != 1 || authors[0] != "石黑一雄" {
		t.Fatalf("authors = %v", second["authors"])
	}

	encoded := rec.Body.String()
	for _, banned := range []string{"41077357", "weread", "reading_time", "progress_percent", "external", "note", "rating", "Dragon Maid"} {
		if strings.Contains(encoded, banned) {
			t.Fatalf("public books payload leaks %q", banned)
		}
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("books part needs a validator")
	}
	// A later media change must rebuild the snapshot despite the 24h TTL.
	moreWork := seedWork(t, db, "book")
	more := seedItem(t, db, moreWork, "book", "第三本书")
	seedBookProgress(t, db, more, "in_progress", nil, &recent)
	if err := db.Exec(`update tb_media_progress set hidden_from_continue = true where media_item_id = ?`, more).Error; err != nil {
		t.Fatalf("pause book: %v", err)
	}
	bumpMediaRevision(t, db)
	conditional := httptest.NewRequest(http.MethodGet, "/public/v1/books", nil)
	conditional.Header.Set("If-None-Match", etag)
	rebuilt := httptest.NewRecorder()
	server.ServeHTTP(rebuilt, conditional)
	if rebuilt.Code != http.StatusOK || rebuilt.Header().Get("ETag") == etag {
		t.Fatalf("media revision change did not rebuild books: %d %v", rebuilt.Code, rebuilt.Header())
	}
	var after []map[string]any
	if err := json.Unmarshal(rebuilt.Body.Bytes(), &after); err != nil || len(after) != 3 {
		t.Fatalf("books after second seed = %d (%v)", len(after), err)
	}
	foundPaused := false
	for _, book := range after {
		if book["title"] == "第三本书" && book["status"] == "paused" {
			foundPaused = true
		}
	}
	if !foundPaused {
		t.Fatalf("hidden in-progress book is not marked paused: %v", after)
	}

	if err := db.Exec(`update tb_media_items set title = ' ' where id = ?`, more).Error; err != nil {
		t.Fatalf("seed invalid book: %v", err)
	}
	bumpMediaRevision(t, db)
	if got := publicGet(t, server, "/public/v1/books"); got.Code != http.StatusInternalServerError {
		t.Fatalf("invalid books response = %d, want 500", got.Code)
	}
	if got := publicGet(t, server, "/public/v1/summary"); got.Code != http.StatusOK {
		t.Fatalf("invalid book broke activity summary: %d %s", got.Code, got.Body.String())
	}
}
