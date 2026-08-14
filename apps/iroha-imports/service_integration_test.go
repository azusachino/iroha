//go:build integration

package imports

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-providers/anilist"
	"github.com/azusachino/iroha/apps/iroha-providers/bangumi"
	"github.com/azusachino/iroha/apps/iroha-providers/parsers"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestIntegrationDailyMetricPersistsAndReprocesses(t *testing.T) {
	db := openImportsIntegrationDB(t)
	rawFileID := uuid.New()
	jobID := uuid.New()
	metric := parsers.DailyMetricObservation{
		Day:    time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		Metric: parsers.DailyMetricRestingHR,
		Value:  57.5,
		Unit:   "count/min",
		Source: "Watch",
	}
	now := time.Now().UTC()

	if err := db.Create(&models.RawFile{
		ID:               rawFileID,
		SHA256:           "body-vitals-integration-" + rawFileID.String(),
		OriginalFilename: "body-vitals.zip",
		StoragePath:      "/tmp/body-vitals.zip",
		SourceKind:       parsers.KindAppleHealthExport,
		UploadedVia:      "integration",
		CreatedAt:        now,
	}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("delete from tb_apple_source_items where source_key = ?", dailyMetricSourceKey(metric)).Error
		_ = db.Exec("delete from tb_daily_metrics where first_raw_file_id = ?", rawFileID).Error
		_ = db.Exec("delete from tb_import_snapshots where raw_file_id = ?", rawFileID).Error
		_ = db.Exec("delete from tb_import_jobs where id = ?", jobID).Error
		_ = db.Exec("delete from tb_raw_files where id = ?", rawFileID).Error
	})
	if err := db.Create(&models.ImportJob{
		ID:            jobID,
		RawFileID:     rawFileID,
		Status:        StatusCompleted,
		ParserKind:    parsers.KindAppleHealthExport,
		ParserVersion: DefaultParserVersion,
		CreatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("create import job: %v", err)
	}

	snapshot1 := models.ImportSnapshot{ID: uuid.New(), ImportJobID: jobID, RawFileID: rawFileID, SHA256: "snapshot-1", ParserVersion: DefaultParserVersion, CreatedAt: now}
	if err := db.Create(&snapshot1).Error; err != nil {
		t.Fatalf("create first snapshot: %v", err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return (&Service{}).persistDailyMetric(tx, models.RawFile{ID: rawFileID}, metric, snapshot1.ID)
	}); err != nil {
		t.Fatalf("persist first metric: %v", err)
	}

	snapshot2 := models.ImportSnapshot{ID: uuid.New(), ImportJobID: jobID, RawFileID: rawFileID, SHA256: "snapshot-2", ParserVersion: DefaultParserVersion, CreatedAt: now.Add(time.Second)}
	if err := db.Create(&snapshot2).Error; err != nil {
		t.Fatalf("create second snapshot: %v", err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return (&Service{}).persistDailyMetric(tx, models.RawFile{ID: rawFileID}, metric, snapshot2.ID)
	}); err != nil {
		t.Fatalf("reconcile unchanged metric: %v", err)
	}
	var metricCount int64
	if err := db.Model(&models.DailyMetric{}).Where("first_raw_file_id = ?", rawFileID).Count(&metricCount).Error; err != nil {
		t.Fatalf("count reconciled metrics: %v", err)
	}
	if metricCount != 1 {
		t.Fatalf("reconciled metric count = %d, want 1", metricCount)
	}

	snapshot3 := models.ImportSnapshot{ID: uuid.New(), ImportJobID: jobID, RawFileID: rawFileID, SHA256: "snapshot-3", ParserVersion: DefaultParserVersion, CreatedAt: now.Add(2 * time.Second)}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := purgeDerivedForRawFile(tx, rawFileID); err != nil {
			return err
		}
		if err := tx.Create(&snapshot3).Error; err != nil {
			return err
		}
		return (&Service{}).persistDailyMetric(tx, models.RawFile{ID: rawFileID}, metric, snapshot3.ID)
	}); err != nil {
		t.Fatalf("reprocess metric: %v", err)
	}
	if err := db.Model(&models.DailyMetric{}).Where("first_raw_file_id = ?", rawFileID).Count(&metricCount).Error; err != nil {
		t.Fatalf("count reprocessed metrics: %v", err)
	}
	if metricCount != 1 {
		t.Fatalf("reprocessed metric count = %d, want 1", metricCount)
	}
}

func TestIntegrationMediaPersistsAndReprocesses(t *testing.T) {
	db := openImportsIntegrationDB(t)
	rawFileID := uuid.New()
	jobID := uuid.New()
	now := time.Now().UTC()
	startedAt := now.Add(-time.Hour)
	completedAt := now.Add(-30 * time.Minute)
	media := observations.Media{
		Provider:    "anilist",
		ExternalID:  "media-integration-" + rawFileID.String(),
		MediaType:   "anime",
		Title:       "Integration Media",
		Status:      "completed",
		Progress:    float64Ptr(12),
		Score:       float64Ptr(9),
		StartedAt:   &startedAt,
		CompletedAt: &completedAt,
	}
	if err := db.Create(&models.RawFile{
		ID:               rawFileID,
		SHA256:           "media-integration-" + rawFileID.String(),
		OriginalFilename: "anilist.json",
		StoragePath:      "/tmp/anilist.json",
		SourceKind:       parsers.KindAniList,
		UploadedVia:      "integration",
		CreatedAt:        now,
	}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("delete from tb_media_consumption_events where raw_file_id = ?", rawFileID).Error
		_ = db.Exec("delete from tb_media_list_items where media_item_id in (select id from tb_media_items where title = ?)", media.Title).Error
		_ = db.Exec("delete from tb_media_lists where source_kind = ? and name = ?", media.Provider, media.Provider+" library").Error
		_ = db.Exec("delete from tb_media_external_refs where external_id = ?", media.ExternalID).Error
		_ = db.Exec("delete from tb_media_progress where media_item_id in (select id from tb_media_items where title = ?)", media.Title).Error
		_ = db.Exec("delete from tb_media_items where title = ?", media.Title).Error
		_ = db.Exec("delete from tb_media_works where primary_title = ?", media.Title).Error
		_ = db.Exec("delete from tb_import_snapshots where raw_file_id = ?", rawFileID).Error
		_ = db.Exec("delete from tb_import_jobs where id = ?", jobID).Error
		_ = db.Exec("delete from tb_raw_files where id = ?", rawFileID).Error
	})
	if err := db.Create(&models.ImportJob{
		ID:            jobID,
		RawFileID:     rawFileID,
		Status:        StatusCompleted,
		ParserKind:    parsers.KindAniList,
		ParserVersion: DefaultParserVersion,
		CreatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("create import job: %v", err)
	}

	snapshot1 := models.ImportSnapshot{ID: uuid.New(), ImportJobID: jobID, RawFileID: rawFileID, SHA256: "media-snapshot-1", ParserVersion: DefaultParserVersion, CreatedAt: now}
	if err := (&Service{db: db}).persistMedia(models.RawFile{ID: rawFileID, SourceKind: parsers.KindAniList}, []observations.Media{media}, snapshot1, false); err != nil {
		t.Fatalf("persist media: %v", err)
	}
	var itemCount, eventCount int64
	if err := db.Model(&models.MediaItem{}).Where("title = ?", media.Title).Count(&itemCount).Error; err != nil {
		t.Fatalf("count media items: %v", err)
	}
	if err := db.Model(&models.MediaConsumptionEvent{}).Where("raw_file_id = ?", rawFileID).Count(&eventCount).Error; err != nil {
		t.Fatalf("count media events: %v", err)
	}
	if itemCount != 1 || eventCount != 1 {
		t.Fatalf("persisted media item/event counts = %d/%d, want 1/1", itemCount, eventCount)
	}

	bridgeMedia := observations.Media{
		Provider: "bangumi", ExternalID: "bangumi-" + rawFileID.String(), MediaType: "anime",
		Title: media.Title, Status: media.Status,
		ExternalRefs: []observations.MediaExternalRef{{Provider: "bangumi", ExternalID: "bangumi-" + rawFileID.String()}},
	}
	bridge := TwoHopMediaRefBridge{
		BangumiToMAL: map[string]string{bridgeMedia.ExternalID: "mal-" + rawFileID.String()},
		MALToAniList: map[string]string{"mal-" + rawFileID.String(): media.ExternalID},
	}
	if err := persistMediaObservation(db, models.RawFile{ID: rawFileID, SourceKind: parsers.KindBangumi}, bridgeMedia, bridge); err != nil {
		t.Fatalf("persist bridged Bangumi media: %v", err)
	}
	var bridgedRef models.MediaExternalRef
	if err := db.Where("provider = ? and external_id = ?", bridgeMedia.Provider, bridgeMedia.ExternalID).First(&bridgedRef).Error; err != nil {
		t.Fatalf("find bridged Bangumi ref: %v", err)
	}
	if bridgedRef.ScopeID == uuid.Nil {
		t.Fatal("bridged Bangumi ref has no item scope")
	}
	if err := db.Exec("delete from tb_media_external_refs where provider = ? and external_id = ?", bridgeMedia.Provider, bridgeMedia.ExternalID).Error; err != nil {
		t.Fatalf("cleanup bridged ref: %v", err)
	}

	snapshot2 := models.ImportSnapshot{ID: uuid.New(), ImportJobID: jobID, RawFileID: rawFileID, SHA256: "media-snapshot-2", ParserVersion: "media-reprocess", CreatedAt: now.Add(time.Second)}
	if err := (&Service{db: db}).persistMedia(models.RawFile{ID: rawFileID, SourceKind: parsers.KindAniList}, []observations.Media{media}, snapshot2, true); err != nil {
		t.Fatalf("reprocess media: %v", err)
	}
	if err := db.Model(&models.MediaItem{}).Where("title = ?", media.Title).Count(&itemCount).Error; err != nil {
		t.Fatalf("count reprocessed media items: %v", err)
	}
	if err := db.Model(&models.MediaConsumptionEvent{}).Where("raw_file_id = ?", rawFileID).Count(&eventCount).Error; err != nil {
		t.Fatalf("count reprocessed media events: %v", err)
	}
	if itemCount != 1 || eventCount != 1 {
		t.Fatalf("reprocessed media item/event counts = %d/%d, want 1/1", itemCount, eventCount)
	}
}

func TestIntegrationMediaRichFieldsAndEventDedup(t *testing.T) {
	db := openImportsIntegrationDB(t)
	svc := &Service{db: db}
	suffix := uuid.New().String()
	idA := "anilist-A-" + suffix
	idB := "anilist-B-" + suffix
	titleA := "Rich Media A " + suffix
	titleB := "Rich Media B " + suffix

	t.Cleanup(func() {
		db.Exec("delete from tb_media_relations where provider = 'anilist' and (from_id in (select id from tb_media_items where title in (?, ?)) or to_id in (select id from tb_media_items where title in (?, ?)))", titleA, titleB, titleA, titleB)
		db.Exec("delete from tb_media_consumption_events where source_event_id like ?", "%"+suffix)
		db.Exec("delete from tb_media_progress where media_item_id in (select id from tb_media_items where title in (?, ?))", titleA, titleB)
		db.Exec("delete from tb_media_external_refs where external_id in (?, ?)", idA, idB)
		db.Exec("delete from tb_media_items where title in (?, ?)", titleA, titleB)
		db.Exec("delete from tb_media_works where primary_title in (?, ?)", titleA, titleB)
		db.Exec("delete from tb_media_lists where source_kind = 'anilist' and name = 'anilist library'")
		db.Exec("delete from tb_import_snapshots where sha256 like ?", "rich-snap-%"+suffix)
		db.Exec("delete from tb_import_jobs where parser_version = ? and raw_file_id in (select id from tb_raw_files where sha256 like ?)", "rich-"+suffix, "rich-%"+suffix)
		db.Exec("delete from tb_raw_files where sha256 like ?", "rich-%"+suffix)
	})

	persist := func(tag string, media observations.Media, reprocess bool) {
		t.Helper()
		rawFileID := uuid.New()
		if err := db.Create(&models.RawFile{ID: rawFileID, SHA256: "rich-" + tag + "-" + suffix, OriginalFilename: "anilist.json", StoragePath: "/tmp/anilist.json", SourceKind: parsers.KindAniList, UploadedVia: "integration", CreatedAt: time.Now().UTC()}).Error; err != nil {
			t.Fatalf("create raw file %s: %v", tag, err)
		}
		jobID := uuid.New()
		if err := db.Create(&models.ImportJob{ID: jobID, RawFileID: rawFileID, Status: StatusCompleted, ParserKind: parsers.KindAniList, ParserVersion: "rich-" + suffix, CreatedAt: time.Now().UTC()}).Error; err != nil {
			t.Fatalf("create import job %s: %v", tag, err)
		}
		snap := models.ImportSnapshot{ID: uuid.New(), ImportJobID: jobID, RawFileID: rawFileID, SHA256: "rich-snap-" + tag + "-" + suffix, ParserVersion: DefaultParserVersion, CreatedAt: time.Now().UTC()}
		if err := svc.persistMedia(models.RawFile{ID: rawFileID, SourceKind: parsers.KindAniList}, []observations.Media{media}, snap, reprocess); err != nil {
			t.Fatalf("persist %s: %v", tag, err)
		}
	}

	mediaA := func(position float64) observations.Media {
		return observations.Media{
			Provider: "anilist", ExternalID: idA, MediaType: "anime_season", Title: titleA, Status: "in_progress",
			Description:   "Original synopsis " + suffix,
			ExternalRefs:  []observations.MediaExternalRef{{Provider: "anilist", ExternalID: idA, MatchedBy: "provider_id"}},
			ProgressState: &observations.MediaProgress{Status: "in_progress", Unit: "episodes", Position: float64Ptr(position), PlayCount: 2},
			Events:        []observations.MediaEvent{{EventType: "list_state", SourceEventID: "entry-A-" + suffix, Unit: "episodes", Position: float64Ptr(position)}},
		}
	}
	workDescription := func() string {
		t.Helper()
		var work models.MediaWork
		if err := db.Joins("join tb_media_items on tb_media_items.work_id = tb_media_works.id").Where("tb_media_items.title = ?", titleA).First(&work).Error; err != nil {
			t.Fatalf("load work: %v", err)
		}
		return work.Description
	}

	// First sync: item + progress (rich fields) + one event.
	persist("a1", mediaA(12), false)
	var progress models.MediaProgress
	if err := db.Joins("join tb_media_items on tb_media_items.id = tb_media_progress.media_item_id").Where("tb_media_items.title = ?", titleA).First(&progress).Error; err != nil {
		t.Fatalf("load progress: %v", err)
	}
	if progress.Unit != "episodes" || progress.PlayCount != 2 {
		t.Fatalf("progress unit/play_count = %q/%d, want episodes/2 (ProgressState dropped)", progress.Unit, progress.PlayCount)
	}
	if got := workDescription(); got != "Original synopsis "+suffix {
		t.Fatalf("work description after first sync = %q, want the synced description (description lives on tb_media_works, not tb_media_items)", got)
	}
	countEvents := func() int64 {
		var n int64
		db.Model(&models.MediaConsumptionEvent{}).Where("source_event_id = ?", "entry-A-"+suffix).Count(&n)
		return n
	}
	if got := countEvents(); got != 1 {
		t.Fatalf("events after first sync = %d, want 1", got)
	}
	var storedEvent models.MediaConsumptionEvent
	if err := db.Where("source_event_id = ?", "entry-A-"+suffix).First(&storedEvent).Error; err != nil {
		t.Fatalf("load first media event: %v", err)
	}
	if storedEvent.EventAt != nil {
		t.Fatalf("unknown list_state event_at = %v, want NULL", storedEvent.EventAt)
	}
	baseEvent := observations.MediaEvent{
		EventType: "list_state", SourceEventID: "entry-A-" + suffix,
		Unit: "episodes", Position: float64Ptr(12),
	}
	unchanged, err := latestEventUnchanged(db, storedEvent.MediaItemID, "anilist", baseEvent)
	if err != nil || !unchanged {
		t.Fatalf("unchanged event comparison = %v, %v; want true, nil", unchanged, err)
	}
	changedEvents := []struct {
		name   string
		mutate func(*observations.MediaEvent)
	}{
		{name: "event_at", mutate: func(event *observations.MediaEvent) {
			at := time.Date(2026, time.August, 12, 10, 0, 0, 0, time.UTC)
			event.EventAt = &at
		}},
		{name: "unit", mutate: func(event *observations.MediaEvent) { event.Unit = "chapters" }},
		{name: "total", mutate: func(event *observations.MediaEvent) { event.Total = float64Ptr(24) }},
		{name: "progress_percent", mutate: func(event *observations.MediaEvent) { event.ProgressPercent = float64Ptr(50) }},
		{name: "rating", mutate: func(event *observations.MediaEvent) { event.Rating = float64Ptr(8) }},
		{name: "rating_scale", mutate: func(event *observations.MediaEvent) { event.RatingScale = float64Ptr(10) }},
		{name: "note", mutate: func(event *observations.MediaEvent) { event.Note = "changed" }},
	}
	for _, test := range changedEvents {
		t.Run("changed_"+test.name, func(t *testing.T) {
			event := baseEvent
			test.mutate(&event)
			unchanged, err := latestEventUnchanged(db, storedEvent.MediaItemID, "anilist", event)
			if err != nil {
				t.Fatalf("compare changed event: %v", err)
			}
			if unchanged {
				t.Fatal("changed event was treated as unchanged")
			}
		})
	}

	// Re-sync unchanged (new raw file, not reprocess): must NOT append a duplicate event.
	persist("a2", mediaA(12), false)
	if got := countEvents(); got != 1 {
		t.Fatalf("events after unchanged re-sync = %d, want 1 (dedup failed)", got)
	}

	// Re-sync with real progress change: append a new history point.
	persist("a3", mediaA(13), false)
	if got := countEvents(); got != 2 {
		t.Fatalf("events after changed re-sync = %d, want 2", got)
	}

	// M2: the owning provider re-syncing a corrected core field overwrites it.
	corrected := mediaA(13)
	corrected.MediaType = "anime"
	corrected.Description = "Corrected synopsis " + suffix
	persist("a4", corrected, false)
	var item models.MediaItem
	if err := db.Where("title = ?", titleA).First(&item).Error; err != nil {
		t.Fatalf("load item: %v", err)
	}
	if item.MediaType != "anime" {
		t.Fatalf("item media_type after owner re-sync = %q, want anime (M2 refresh)", item.MediaType)
	}
	if got := workDescription(); got != "Corrected synopsis "+suffix {
		t.Fatalf("work description after owner re-sync = %q, want the corrected description (M2 refresh)", got)
	}

	// Relation: B -> A. Both endpoints resolve, so the edge must persist.
	mediaB := observations.Media{
		Provider: "anilist", ExternalID: idB, MediaType: "anime_season", Title: titleB, Status: "planned",
		ExternalRefs: []observations.MediaExternalRef{{Provider: "anilist", ExternalID: idB, MatchedBy: "provider_id"}},
		Relations:    []observations.MediaRelation{{FromType: "item", FromExternalID: idB, ToType: "item", ToExternalID: idA, RelationType: "SEQUEL", Provider: "anilist"}},
	}
	persist("b1", mediaB, false)
	var relationCount int64
	if err := db.Model(&models.MediaRelation{}).Where("relation_type = 'SEQUEL' and to_id in (select id from tb_media_items where title = ?)", titleA).Count(&relationCount).Error; err != nil {
		t.Fatalf("count relations: %v", err)
	}
	if relationCount != 1 {
		t.Fatalf("SEQUEL relations persisted = %d, want 1 (tb_media_relations not written)", relationCount)
	}
	// Idempotent: re-syncing B must not duplicate the edge.
	persist("b2", mediaB, false)
	db.Model(&models.MediaRelation{}).Where("relation_type = 'SEQUEL' and to_id in (select id from tb_media_items where title = ?)", titleA).Count(&relationCount)
	if relationCount != 1 {
		t.Fatalf("SEQUEL relations after re-sync = %d, want 1 (edge duplicated)", relationCount)
	}
}

func TestIntegrationProviderMediaEventsPreserveTimestamps(t *testing.T) {
	db := openImportsIntegrationDB(t)
	svc := &Service{db: db}
	suffix := uuid.New().String()

	persist := func(t *testing.T, sourceKind string, media observations.Media) (models.MediaConsumptionEvent, models.MediaProgress) {
		t.Helper()
		rawFileID := uuid.New()
		jobID := uuid.New()
		now := time.Now().UTC()
		if err := db.Create(&models.RawFile{ID: rawFileID, SHA256: "provider-events-" + suffix + sourceKind, OriginalFilename: sourceKind + ".json", StoragePath: "/tmp/" + sourceKind + ".json", SourceKind: sourceKind, UploadedVia: "integration", CreatedAt: now}).Error; err != nil {
			t.Fatalf("create raw file: %v", err)
		}
		if err := db.Create(&models.ImportJob{ID: jobID, RawFileID: rawFileID, Status: StatusCompleted, ParserKind: sourceKind, ParserVersion: "provider-events-" + suffix, CreatedAt: now}).Error; err != nil {
			t.Fatalf("create import job: %v", err)
		}
		snapshot := models.ImportSnapshot{ID: uuid.New(), ImportJobID: jobID, RawFileID: rawFileID, SHA256: "provider-events-snapshot-" + suffix + sourceKind, ParserVersion: DefaultParserVersion, CreatedAt: now}
		if err := svc.persistMedia(models.RawFile{ID: rawFileID, SourceKind: sourceKind}, []observations.Media{media}, snapshot, false); err != nil {
			t.Fatalf("persist provider media: %v", err)
		}
		var ref models.MediaExternalRef
		if err := db.Where("provider = ? and external_id = ?", media.Provider, media.ExternalID).First(&ref).Error; err != nil {
			t.Fatalf("load provider ref: %v", err)
		}
		var event models.MediaConsumptionEvent
		if err := db.Where("media_item_id = ?", ref.ScopeID).First(&event).Error; err != nil {
			t.Fatalf("load provider event: %v", err)
		}
		var progress models.MediaProgress
		if err := db.Where("media_item_id = ?", ref.ScopeID).First(&progress).Error; err != nil {
			t.Fatalf("load provider progress: %v", err)
		}
		t.Cleanup(func() {
			db.Exec("delete from tb_media_consumption_events where raw_file_id = ?", rawFileID)
			db.Exec("delete from tb_media_list_items where media_item_id = ?", ref.ScopeID)
			db.Exec("delete from tb_media_progress where media_item_id = ?", ref.ScopeID)
			db.Exec("delete from tb_media_external_refs where id = ?", ref.ID)
			db.Exec("delete from tb_media_items where id = ?", ref.ScopeID)
			db.Exec("delete from tb_media_works where primary_title = ?", media.Title)
			db.Exec("delete from tb_import_snapshots where raw_file_id = ?", rawFileID)
			db.Exec("delete from tb_import_jobs where id = ?", jobID)
			db.Exec("delete from tb_raw_files where id = ?", rawFileID)
		})
		return event, progress
	}

	aniEntries, err := anilist.ParseSnapshot([]byte(fmt.Sprintf(`{
  "data": {"MediaListCollection": {"lists": [{"entries": [{
    "id": 901, "status": "COMPLETED", "completedAt": {"year": 2026, "month": 8, "day": 10},
    "media": {"id": 901, "type": "ANIME", "format": "TV", "title": {"romaji": "Ani %s"}}
  }]}]}}
}`, suffix)))
	if err != nil {
		t.Fatalf("parse AniList provider fixture: %v", err)
	}
	aniEvent, aniProgress := persist(t, parsers.KindAniList, aniEntries[0])
	if aniEvent.EventAt != nil {
		t.Fatalf("AniList list_state event_at = %v, want NULL", aniEvent.EventAt)
	}
	if aniProgress.FinishedAt == nil || aniProgress.FinishedAt.Format("2006-01-02") != "2026-08-10" {
		t.Fatalf("AniList finished_at = %v, want 2026-08-10", aniProgress.FinishedAt)
	}

	bangumiEntries, err := bangumi.ParseSnapshot([]byte(fmt.Sprintf(`{
  "total": 1, "data": [{"subject_type": 2, "type": 2, "updated_at": "2026-08-12T10:00:00+00:00",
    "subject": {"id": 902, "name": "Bangumi %s"}}]
}`, suffix)))
	if err != nil {
		t.Fatalf("parse Bangumi provider fixture: %v", err)
	}
	bgEvent, bgProgress := persist(t, parsers.KindBangumi, bangumiEntries[0])
	if bgEvent.EventAt != nil {
		t.Fatalf("Bangumi list_state event_at = %v, want NULL", bgEvent.EventAt)
	}
	if bgProgress.Status != "completed" || bgProgress.FinishedAt != nil {
		t.Fatalf("Bangumi completion projection = status %q finished_at %v, want completed/NULL", bgProgress.Status, bgProgress.FinishedAt)
	}
}

func float64Ptr(value float64) *float64 { return &value }

func openImportsIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get integration db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}
