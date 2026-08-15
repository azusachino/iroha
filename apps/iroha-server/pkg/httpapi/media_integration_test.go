//go:build integration

package httpapi

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TestMediaAggregatesIntegration exercises the real 3-query aggregates SQL
// against Postgres: completion year rollup (via day-precision progress OR a
// finished/completed event), score-distribution normalization across differing
// rating scales, type split by work_kind, and the weighted average / this-year
// totals computed in Go.
func TestMediaAggregatesIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)

	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	animeWork := seedWork(t, db, "anime")
	bookWork := seedWork(t, db, "book")

	// A: anime, completed via a day-precision progress fact this year, rated 8/10.
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
	if result.Totals.CurrentCompletedCount != 2 {
		t.Fatalf("current_completed_count = %d, want 2", result.Totals.CurrentCompletedCount)
	}
	if result.Totals.ThisYearCompleted != 2 {
		t.Fatalf("this_year_completed = %d, want 2", result.Totals.ThisYearCompleted)
	}
	// Weighted average of normalized ratings: (8 + 8 + 3) / 3 = 6.333...
	if diff := result.Totals.AverageRating - 19.0/3.0; diff > 1e-6 || diff < -1e-6 {
		t.Fatalf("average_rating = %v, want %v", result.Totals.AverageRating, 19.0/3.0)
	}
}

func TestMediaAggregatesFiltersIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)

	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	animeWork := seedWork(t, db, "anime")
	bookWork := seedWork(t, db, "book")
	animeCompleted := seedItem(t, db, animeWork, "anime_season", "Anime completed")
	seedProgress(t, db, animeCompleted, "completed", ptrTime(time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)))
	animeActive := seedItem(t, db, animeWork, "movie", "Anime active")
	seedProgress(t, db, animeActive, "in_progress", nil)
	bookCompleted := seedItem(t, db, bookWork, "book", "Book completed")
	seedProgress(t, db, bookCompleted, "completed", ptrTime(time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)))

	service := media.NewService(db)
	family, err := service.AggregatesFiltered(now, media.ListFilters{Family: "anime"})
	if err != nil {
		t.Fatalf("anime aggregates: %v", err)
	}
	if family.Totals.ItemCount != 2 || family.Totals.CompletedCount != 1 || family.Totals.CurrentCompletedCount != 1 {
		t.Fatalf("anime totals = %+v, want 2 items and 1 completed", family.Totals)
	}

	completed, err := service.AggregatesFiltered(now, media.ListFilters{Status: "completed"})
	if err != nil {
		t.Fatalf("completed aggregates: %v", err)
	}
	if completed.Totals.ItemCount != 2 || completed.Totals.CompletedCount != 2 || completed.Totals.CurrentCompletedCount != 2 {
		t.Fatalf("completed totals = %+v, want 2 items and 2 completed", completed.Totals)
	}

	year := 2026
	yearOnly, err := service.AggregatesFiltered(now, media.ListFilters{CompletedYear: &year})
	if err != nil {
		t.Fatalf("year aggregates: %v", err)
	}
	if yearOnly.Totals.ItemCount != 2 || yearOnly.Totals.CompletedCount != 2 || yearOnly.Totals.CurrentCompletedCount != 2 {
		t.Fatalf("year totals = %+v, want 2 items and 2 completed", yearOnly.Totals)
	}

	empty, err := service.AggregatesFiltered(now, media.ListFilters{Family: "game"})
	if err != nil {
		t.Fatalf("empty game aggregates: %v", err)
	}
	if empty.CompletionsByYear == nil || empty.ScoreDistribution == nil || empty.TypeSplit == nil {
		t.Fatal("empty aggregate arrays must be non-nil")
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

	// Fuzzy/undated completions are excluded from the day-scoped report.
	itemUndated := seedItem(t, db, work, "game", "Undated")
	seedProgress(t, db, itemUndated, "completed", nil)
	itemMonth := seedItem(t, db, work, "book", "Month-only")
	if err := db.Exec(`insert into tb_media_progress (media_item_id, status, completed_on_value, completed_on_precision, updated_at)
		values (?, 'completed', ?, 'month', ?)`, itemMonth, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed month-precision progress: %v", err)
	}
	itemYear := seedItem(t, db, work, "book", "Year-only")
	if err := db.Exec(`insert into tb_media_progress (media_item_id, status, completed_on_value, completed_on_precision, updated_at)
		values (?, 'completed', ?, 'year', ?)`, itemYear, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed year-precision progress: %v", err)
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

func TestMediaEventTypeConstraintIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)
	work := seedWork(t, db, "media")
	item := seedItem(t, db, work, "book", "Constrained event")
	err := db.Exec(`insert into tb_media_consumption_events
		(id, media_item_id, event_type, event_at, source_kind, created_at)
		values (?, ?, 'unknown_state', ?, 'test', ?)`, uuid.New(), item,
		time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC), time.Now().UTC()).Error
	if err == nil {
		t.Fatal("database accepted an event type outside the canonical vocabulary")
	}
}

func TestMediaEventsIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)

	from := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	work := seedWork(t, db, "media")
	item := seedItem(t, db, work, "manga", "A")

	if err := db.Exec(`insert into tb_media_consumption_events
		(id, media_item_id, event_type, event_at, source_kind, created_at)
		values (?, ?, 'rewatched', ?, 'manual', ?)`, uuid.New(), item,
		time.Date(2026, 8, 14, 13, 0, 0, 0, time.UTC), time.Now().UTC()).Error; err != nil {
		t.Fatalf("seed media event: %v", err)
	}

	result, err := media.NewService(db).Events(media.EventListFilters{
		From: &from, To: &to, Limit: 100,
	})
	if err != nil {
		t.Fatalf("list media events: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].EventType != "rewatched" {
		t.Fatalf("events = %+v, want only the dated user event", result.Items)
	}
}

func TestMediaDetailIncludesProviderUpdatesSeparately(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)
	work := seedWork(t, db, "media")
	item := seedItem(t, db, work, "manga", "Provider activity detail")
	seedProgress(t, db, item, "in_progress", nil)
	effectiveAt := time.Date(2026, 8, 15, 2, 6, 0, 0, time.UTC)
	observedAt := effectiveAt.Add(2 * time.Minute)
	if err := db.Exec(`insert into tb_media_state_history
		(id, media_item_id, source_kind, source_event_id, observed_at, time_basis,
		 change_kind, state_fingerprint, status, created_at)
		values (?, ?, 'anilist', 'snapshot-1', ?, 'iroha_observed', 'snapshot', ?, 'in_progress', ?)`,
		uuid.New(), item, observedAt, "detail-observation", observedAt).Error; err != nil {
		t.Fatalf("seed observation snapshot: %v", err)
	}
	if err := db.Exec(`insert into tb_media_state_history
		(id, media_item_id, source_kind, source_event_id, observed_at, effective_at,
		 time_basis, change_kind, state_fingerprint, status, unit, position, note, created_at)
		values (?, ?, 'anilist', 'activity-1', ?, ?, 'provider_activity',
		 'provider_activity', ?, 'in_progress', 'chapters', 210, 'read chapter 210', ?)`,
		uuid.New(), item, observedAt, effectiveAt, "detail-provider-activity", observedAt).Error; err != nil {
		t.Fatalf("seed provider activity: %v", err)
	}

	detail, found, err := media.NewService(db).Get(item)
	if err != nil {
		t.Fatalf("get media detail: %v", err)
	}
	if !found {
		t.Fatal("media detail not found")
	}
	if len(detail.Events) != 0 {
		t.Fatalf("exact events = %d, want no exact events", len(detail.Events))
	}
	if len(detail.Updates) != 1 || detail.Updates[0].Position == nil || *detail.Updates[0].Position != 210 {
		t.Fatalf("provider updates = %+v, want one chapter-210 update", detail.Updates)
	}
}

func TestMediaExactEventIdempotencyIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)
	work := seedWork(t, db, "media")
	item := seedItem(t, db, work, "book", "Idempotent event")
	at := time.Date(2026, 8, 15, 10, 30, 0, 0, time.UTC)
	input := media.CreateEventInput{
		MediaItemID: item, EventType: "read", EventAt: at,
		SourceKind: "local_agent", SourceEventID: "capture-1", Unit: "pages",
		Position: floatPtr(12), Note: "exact playback checkpoint",
	}

	first, err := media.NewService(db).CreateEvent(input)
	if err != nil {
		t.Fatalf("create exact event: %v", err)
	}
	second, err := media.NewService(db).CreateEvent(input)
	if err != nil {
		t.Fatalf("retry exact event: %v", err)
	}
	if first.ID != second.ID || !second.OccurredAt.Equal(at) {
		t.Fatalf("retry = %+v, want the original event %s at %s", second, first.ID, at)
	}

	input.Note = "different payload"
	if _, err := media.NewService(db).CreateEvent(input); err != media.ErrEventConflict {
		t.Fatalf("conflicting retry error = %v, want %v", err, media.ErrEventConflict)
	}
	var count int64
	if err := db.Table("tb_media_consumption_events").Where("source_kind = ? and source_event_id = ?", "local_agent", "capture-1").Count(&count).Error; err != nil {
		t.Fatalf("count exact events: %v", err)
	}
	if count != 1 {
		t.Fatalf("exact event count = %d, want 1", count)
	}
}

func TestMediaExactEventHTTPIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)
	work := seedWork(t, db, "media")
	item := seedItem(t, db, work, "book", "HTTP event")
	server := newIntegrationServer(t, db)
	body := `{"media_id":"` + ids.Encode(ids.MediaPrefix, item) + `","event_type":"read","event_at":"2026-08-15T10:30:00Z","idempotency_key":"http-capture-1","unit":"pages","position":12}`

	first := requestJSON(t, server, http.MethodPost, "/api/v1/media/events", body, http.StatusOK, nil)
	second := requestJSON(t, server, http.MethodPost, "/api/v1/media/events", body, http.StatusOK, nil)
	if first["id"] != second["id"] || first["event_at"] != "2026-08-15T10:30:00Z" {
		t.Fatalf("HTTP event retry = %#v/%#v, want one canonical event", first, second)
	}

	conflict := strings.Replace(body, `"position":12`, `"position":13`, 1)
	requestJSON(t, server, http.MethodPost, "/api/v1/media/events", conflict, http.StatusConflict, nil)
	requestJSON(t, server, http.MethodPost, "/api/v1/media/events", `{"media_id":"`+ids.Encode(ids.MediaPrefix, item)+`","event_type":"read","idempotency_key":"missing-time"}`, http.StatusBadRequest, nil)
}

func TestMediaDatedChangesUseEffectiveDateIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	resetMediaTables(t, db)
	work := seedWork(t, db, "media")
	item := seedItem(t, db, work, "manga", "Dated manga update")
	observedAfterDate := time.Date(2026, 8, 16, 2, 0, 0, 0, time.UTC)
	sourceDate := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	providerActivity := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	if err := db.Exec(`insert into tb_media_state_history
		(id, media_item_id, source_kind, source_event_id, observed_at, time_basis, change_kind,
		 state_fingerprint, status, effective_on_value, effective_on_precision, created_at)
		values (?, ?, 'anilist', 'entry-1', ?, 'source_date', 'changed', ?, 'completed', ?, 'day', ?)`,
		uuid.New(), item, observedAfterDate, "source-date-fingerprint", sourceDate, observedAfterDate).Error; err != nil {
		t.Fatalf("seed source-date change: %v", err)
	}
	if err := db.Exec(`insert into tb_media_state_history
		(id, media_item_id, source_kind, source_event_id, observed_at, effective_at, time_basis, change_kind,
		 state_fingerprint, status, created_at)
		values (?, ?, 'anilist', 'activity-1', ?, ?, 'provider_activity', 'provider_activity', ?, 'in_progress', ?)`,
		uuid.New(), item, observedAfterDate, providerActivity, "provider-activity-fingerprint", observedAfterDate).Error; err != nil {
		t.Fatalf("seed provider activity: %v", err)
	}
	if err := db.Exec(`insert into tb_media_state_history
		(id, media_item_id, source_kind, source_event_id, observed_at, time_basis, change_kind,
		 state_fingerprint, status, created_at)
		values (?, ?, 'bangumi', 'snapshot-1', ?, 'iroha_observed', 'snapshot', ?, 'in_progress', ?)`,
		uuid.New(), item, sourceDate, "observed-only-fingerprint", sourceDate).Error; err != nil {
		t.Fatalf("seed observed-only change: %v", err)
	}

	page, err := media.NewService(db).DatedChanges(sourceDate, sourceDate.Add(24*time.Hour), 100)
	if err != nil {
		t.Fatalf("dated changes: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("dated changes = %d, want source-date and provider-activity rows", len(page.Items))
	}
	for _, change := range page.Items {
		if change.TimeBasis == "iroha_observed" {
			t.Fatalf("observation-only row entered dated changes: %+v", change)
		}
	}
}

func resetMediaTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`truncate table
		tb_media_consumption_events,
		tb_media_state_history,
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

func seedProgress(t *testing.T, db *gorm.DB, itemID uuid.UUID, status string, completedOn *time.Time) {
	t.Helper()
	precision := ""
	if completedOn != nil {
		precision = "day"
	}
	if err := db.Exec(`insert into tb_media_progress (media_item_id, status, completed_on_value, completed_on_precision, updated_at)
		values (?, ?, ?, ?, ?)`, itemID, status, completedOn, precision, time.Now().UTC()).Error; err != nil {
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

func floatPtr(value float64) *float64 { return &value }
