//go:build integration

package httpapi

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TestMediaAggregatesIntegration exercises the real 3-query aggregates SQL
// against Postgres: completion year rollup (via progress.finished_at OR a
// finished/completed event), score-distribution normalization across differing
// rating scales, type split by work_kind, and the weighted average / this-year
// totals computed in Go.
func TestMediaAggregatesIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)

	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	animeWork := seedWork(t, db, "anime")
	bookWork := seedWork(t, db, "book")

	// A: anime, completed via progress.finished_at this year, rated 8/10.
	itemA := seedItem(t, db, animeWork, "anime_season", "A")
	seedProgress(t, db, itemA, "completed", ptrTime(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)))
	seedRatingEvent(t, db, itemA, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), 8, 10)

	// B: anime, completed via a 'finished' event last year, rated 4/5 -> 8.0.
	itemB := seedItem(t, db, animeWork, "anime_season", "B")
	seedProgress(t, db, itemB, "in_progress", nil)
	seedFinishEvent(t, db, itemB, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	seedRatingEvent(t, db, itemB, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), 4, 5)

	// C: book, completed this year, rated 3/10.
	itemC := seedItem(t, db, bookWork, "book", "C")
	seedProgress(t, db, itemC, "completed", ptrTime(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)))
	seedRatingEvent(t, db, itemC, time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), 3, 10)

	// D: book, in progress, unrated, never finished.
	itemD := seedItem(t, db, bookWork, "book", "D")
	seedProgress(t, db, itemD, "in_progress", nil)

	result, err := media.NewService(db).Aggregates(now)
	if err != nil {
		t.Fatalf("aggregates: %v", err)
	}

	// Completions by year: 2025 -> B, 2026 -> A + C.
	byYear := map[int]int{}
	for _, bucket := range result.CompletionsByYear {
		byYear[bucket.Year] = bucket.Count
	}
	if byYear[2025] != 1 || byYear[2026] != 2 {
		t.Fatalf("completions_by_year = %+v, want 2025:1 2026:2", result.CompletionsByYear)
	}

	// Score distribution: normalized 8,8,3 -> {3:1, 8:2}.
	byScore := map[float64]int{}
	for _, bucket := range result.ScoreDistribution {
		byScore[bucket.Score] = bucket.Count
	}
	if byScore[8] != 2 || byScore[3] != 1 {
		t.Fatalf("score_distribution = %+v, want 8:2 3:1", result.ScoreDistribution)
	}

	// Type split groups by item media_type: anime_season (A,B) and book (C,D).
	byType := map[string]int{}
	for _, bucket := range result.TypeSplit {
		byType[bucket.Type] = bucket.Count
	}
	if byType["anime_season"] != 2 || byType["book"] != 2 {
		t.Fatalf("type_split = %+v, want anime_season:2 book:2", result.TypeSplit)
	}

	if result.Totals.ItemCount != 4 {
		t.Fatalf("item_count = %d, want 4", result.Totals.ItemCount)
	}
	if result.Totals.CompletedCount != 3 {
		t.Fatalf("completed_count = %d, want 3", result.Totals.CompletedCount)
	}
	if result.Totals.ThisYearCompleted != 2 {
		t.Fatalf("this_year_completed = %d, want 2", result.Totals.ThisYearCompleted)
	}
	// Weighted average of normalized ratings: (8 + 8 + 3) / 3 = 6.333...
	if diff := result.Totals.AverageRating - 19.0/3.0; diff > 1e-6 || diff < -1e-6 {
		t.Fatalf("average_rating = %v, want %v", result.Totals.AverageRating, 19.0/3.0)
	}
}

func TestMediaPeriodReportIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)

	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	work := seedWork(t, db, "media")
	itemA := seedItem(t, db, work, "anime_season", "A")
	itemB := seedItem(t, db, work, "book", "B")

	// A has two dated completion events in the month. The report emits one
	// completion, using the latest dated candidate for that item.
	seedRatingEvent(t, db, itemA, time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC), 4, 5)
	seedFinishEvent(t, db, itemA, time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC))
	if err := db.Exec(`insert into tb_media_consumption_events
		(id, media_item_id, event_type, event_at, source_kind, created_at)
		values (?, ?, 'completed', ?, 'test', ?)`, uuid.New(), itemA,
		time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC), time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed completed event: %v", err)
	}

	// B has a dated progress completion but no consumption event.
	seedProgress(t, db, itemB, "completed", ptrTime(time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)))

	// Undated completions are excluded, even when they were synchronized in the
	// selected month. list_state is a dated provider snapshot, not activity.
	itemUndated := seedItem(t, db, work, "game", "Undated")
	if err := db.Exec(`insert into tb_media_consumption_events
		(id, media_item_id, event_type, source_kind, created_at)
		values (?, ?, 'completed', 'test', ?)`, uuid.New(), itemUndated, time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed undated event: %v", err)
	}
	if err := db.Exec(`insert into tb_media_consumption_events
		(id, media_item_id, event_type, event_at, source_kind, created_at)
		values (?, ?, 'list_state', ?, 'test', ?)`, uuid.New(), itemUndated,
		time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC), time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed snapshot event: %v", err)
	}

	result, err := media.NewService(db).PeriodReport(media.PeriodFilters{From: from, To: to})
	if err != nil {
		t.Fatalf("period report: %v", err)
	}
	if result.EventCount != 3 || result.CompletedCount != 2 || result.RatedCount != 1 {
		t.Fatalf("counts = %+v", result)
	}
	if result.AverageRating == nil || *result.AverageRating != 8 {
		t.Fatalf("average_rating = %v, want 8", result.AverageRating)
	}
	if len(result.CompletedItems) != 2 || result.CompletedItems[0].ID != itemA || result.CompletedItems[1].ID != itemB {
		t.Fatalf("completed_items = %+v", result.CompletedItems)
	}
	if len(result.ByKind) != 2 || result.ByKind[0].Kind != "anime_season" || result.ByKind[0].EventCount != 3 || result.ByKind[0].CompletedCount != 1 || result.ByKind[1].Kind != "book" || result.ByKind[1].CompletedCount != 1 {
		t.Fatalf("by_kind = %+v", result.ByKind)
	}
}

func TestMediaEventsIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)

	from := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	work := seedWork(t, db, "media")
	item := seedItem(t, db, work, "manga", "A")

	// A provider list snapshot is synchronization metadata, not a user media
	// event. Its event_at is intentionally NULL, even when the sync happened
	// inside the requested day.
	if err := db.Exec(`insert into tb_media_consumption_events
		(id, media_item_id, event_type, source_kind, created_at)
		values (?, ?, 'list_state', 'anilist', ?)`, uuid.New(), item,
		time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)).Error; err != nil {
		t.Fatalf("seed list snapshot: %v", err)
	}
	if err := db.Exec(`insert into tb_media_consumption_events
		(id, media_item_id, event_type, event_at, source_kind, created_at)
		values (?, ?, 'rewatch', ?, 'manual', ?)`, uuid.New(), item,
		time.Date(2026, 8, 14, 13, 0, 0, 0, time.UTC), time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed media event: %v", err)
	}

	result, err := media.NewService(db).Events(media.EventListFilters{
		From: &from, To: &to, Limit: 100,
	})
	if err != nil {
		t.Fatalf("list media events: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].EventType != "rewatch" {
		t.Fatalf("events = %+v, want only the dated user event", result.Items)
	}
}

func resetMediaTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`truncate table
		tb_media_consumption_events,
		tb_media_progress,
		tb_media_items,
		tb_media_works
		cascade`).Error; err != nil {
		t.Fatalf("reset media tables: %v", err)
	}
}

func seedWork(t *testing.T, db *gorm.DB, kind string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	if err := db.Exec(`insert into tb_media_works (id, work_kind, primary_title, created_at, updated_at)
		values (?, ?, ?, ?, ?)`, id, kind, kind+" work", now, now).Error; err != nil {
		t.Fatalf("seed work: %v", err)
	}
	return id
}

func seedItem(t *testing.T, db *gorm.DB, workID uuid.UUID, mediaType, title string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	if err := db.Exec(`insert into tb_media_items (id, work_id, media_type, item_role, title, created_at, updated_at)
		values (?, ?, ?, 'primary', ?, ?, ?)`, id, workID, mediaType, title, now, now).Error; err != nil {
		t.Fatalf("seed item: %v", err)
	}
	return id
}

func seedProgress(t *testing.T, db *gorm.DB, itemID uuid.UUID, status string, finishedAt *time.Time) {
	t.Helper()
	if err := db.Exec(`insert into tb_media_progress (media_item_id, status, finished_at, updated_at)
		values (?, ?, ?, ?)`, itemID, status, finishedAt, time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed progress: %v", err)
	}
}

func seedFinishEvent(t *testing.T, db *gorm.DB, itemID uuid.UUID, at time.Time) {
	t.Helper()
	if err := db.Exec(`insert into tb_media_consumption_events (id, media_item_id, event_type, event_at, source_kind, created_at)
		values (?, ?, 'finished', ?, 'test', ?)`, uuid.New(), itemID, at, time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed finish event: %v", err)
	}
}

func seedRatingEvent(t *testing.T, db *gorm.DB, itemID uuid.UUID, at time.Time, rating, scale float64) {
	t.Helper()
	if err := db.Exec(`insert into tb_media_consumption_events (id, media_item_id, event_type, event_at, rating, rating_scale, source_kind, created_at)
		values (?, ?, 'rated', ?, ?, ?, 'test', ?)`, uuid.New(), itemID, at, rating, scale, time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed rating event: %v", err)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
