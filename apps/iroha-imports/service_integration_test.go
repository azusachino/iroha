//go:build integration

package imports

import (
	"os"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
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
