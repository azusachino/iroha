//go:build integration

package daily

import (
	"os"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPeriodReportGroupsSparseMetricsAndCountsUnionDays(t *testing.T) {
	db := openDailyIntegrationDB(t)
	rawID := uuid.New()
	createdAt := time.Now().UTC()
	if err := db.Create(&models.RawFile{ID: rawID, SHA256: "daily-period-report-" + rawID.String(), OriginalFilename: "daily.xml", StoragePath: "/tmp/daily.xml", SourceKind: "apple_health_export", UploadedVia: "test", CreatedAt: createdAt}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	t.Cleanup(func() { db.Exec("delete from tb_raw_files where id = ?", rawID) })

	dayOne := time.Date(2026, time.February, 2, 0, 0, 0, 0, time.UTC)
	dayTwo := time.Date(2026, time.February, 3, 0, 0, 0, 0, time.UTC)
	summary := models.DailySummary{ID: uuid.New(), Day: dayOne, MoveKcal: 500, Source: "Watch", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt}
	metrics := []models.DailyMetric{
		{ID: uuid.New(), Day: dayOne, Metric: "steps", Value: 1000, Unit: "count", Source: "Watch", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), Day: dayTwo, Metric: "hrv", Value: 42, Unit: "ms", Source: "Watch", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
	}
	if err := db.Create(&summary).Error; err != nil {
		t.Fatalf("create daily summary: %v", err)
	}
	t.Cleanup(func() { db.Delete(&models.DailySummary{}, "id = ?", summary.ID) })
	for _, metric := range metrics {
		if err := db.Create(&metric).Error; err != nil {
			t.Fatalf("create daily metric: %v", err)
		}
		t.Cleanup(func() { db.Delete(&models.DailyMetric{}, "id = ?", metric.ID) })
	}

	result, err := NewService(db).PeriodReport(PeriodFilters{From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("period report: %v", err)
	}
	if result.ObservedDays != 2 || len(result.MetricAverages) != 2 || result.MetricAverages[0].Metric != "hrv" || result.MetricAverages[1].Metric != "steps" {
		t.Fatalf("daily period report = %+v", result)
	}
}

func openDailyIntegrationDB(t *testing.T) *gorm.DB {
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
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}
