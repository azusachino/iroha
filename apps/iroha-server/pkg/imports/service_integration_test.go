//go:build integration

package imports

import (
	"os"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/parsers"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestIntegrationDailyMetricPersistsAndReprocesses(t *testing.T) {
	db := openImportsIntegrationDB(t)
	rawFileID := uuid.New()
	jobID := uuid.New()
	metric := parsers.ParsedDailyMetric{
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
