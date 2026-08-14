//go:build integration

package activities

import (
	"os"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPeriodReportUsesRequestedHalfOpenWindowAndSportOrder(t *testing.T) {
	db := openActivitiesIntegrationDB(t)
	rawID := uuid.New()
	createdAt := time.Now().UTC()
	if err := db.Create(&models.RawFile{
		ID: rawID, SHA256: "activities-period-report-" + rawID.String(), OriginalFilename: "period.gpx",
		StoragePath: "/tmp/period.gpx", SourceKind: "gpx", UploadedVia: "test", CreatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	t.Cleanup(func() { db.Exec("delete from tb_raw_files where id = ?", rawID) })

	tokyo := time.FixedZone("+09", 9*60*60)
	inside := time.Date(2026, time.February, 1, 0, 0, 0, 0, tokyo)
	upper := time.Date(2026, time.March, 1, 0, 0, 0, 0, tokyo)
	distanceRun := 2500.0
	distanceBike := 1000.0
	runDuration, bikeDuration := 600, 300
	rows := []models.Activity{
		{ID: uuid.New(), SportType: "run", StartedAt: inside, DistanceM: &distanceRun, DurationS: &runDuration, SourceKind: "gpx", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), SportType: "bike", StartedAt: inside.Add(24 * time.Hour), DistanceM: &distanceBike, DurationS: &bikeDuration, SourceKind: "gpx", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), SportType: "run", StartedAt: upper, DistanceM: &distanceRun, DurationS: &runDuration, SourceKind: "gpx", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create activity: %v", err)
		}
		t.Cleanup(func() { db.Exec("delete from tb_activities where id = ?", row.ID) })
	}

	result, err := NewService(db).PeriodReport(PeriodFilters{From: inside, To: upper, Timezone: "Asia/Tokyo"})
	if err != nil {
		t.Fatalf("period report: %v", err)
	}
	if result.Totals.ActivityCount != 2 || result.Totals.DistanceM != 3500 || result.Totals.DistanceKnownCount != 2 || result.Totals.DurationS != 900 {
		t.Fatalf("period totals = %+v", result.Totals)
	}
	if len(result.BySport) != 2 || result.BySport[0].Sport != "bike" || result.BySport[1].Sport != "run" {
		t.Fatalf("sport totals = %+v", result.BySport)
	}
}

func TestSummaryUsesRequestedTimezoneAndKeepsFacetsUsefulForFilters(t *testing.T) {
	db := openActivitiesIntegrationDB(t)
	rawID := uuid.New()
	createdAt := time.Now().UTC()
	if err := db.Create(&models.RawFile{
		ID: rawID, SHA256: "activities-summary-facets-" + rawID.String(), OriginalFilename: "summary.gpx",
		StoragePath: "/tmp/summary.gpx", SourceKind: "gpx", UploadedVia: "test", CreatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	t.Cleanup(func() { db.Exec("delete from tb_raw_files where id = ?", rawID) })

	runJanuary := 1000.0
	runFebruary := 2000.0
	bikeJanuary := 3000.0
	rows := []models.Activity{
		// 15:00 UTC is the following calendar day in Tokyo.
		{ID: uuid.New(), SportType: "run", StartedAt: time.Date(2098, time.December, 31, 15, 0, 0, 0, time.UTC), DistanceM: &runJanuary, SourceKind: "gpx", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), SportType: "bike", StartedAt: time.Date(2098, time.December, 31, 15, 0, 0, 0, time.UTC), DistanceM: &bikeJanuary, SourceKind: "gpx", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), SportType: "run", StartedAt: time.Date(2099, time.January, 31, 15, 0, 0, 0, time.UTC), DistanceM: &runFebruary, SourceKind: "gpx", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create activity: %v", err)
		}
		t.Cleanup(func() { db.Exec("delete from tb_activities where id = ?", row.ID) })
	}

	summary, err := NewService(db).SummaryInTimezone("2099", "run", "Asia/Tokyo")
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.Totals.ActivityCount != 2 || summary.Totals.DistanceM != 3000 {
		t.Fatalf("summary totals = %+v", summary.Totals)
	}
	if len(summary.ByMonth) != 2 || summary.ByMonth[0].Key != "2099-02" || summary.ByMonth[1].Key != "2099-01" {
		t.Fatalf("summary months = %+v", summary.ByMonth)
	}
	if len(summary.BySport) != 2 {
		t.Fatalf("summary sports = %+v", summary.BySport)
	}
	for _, bucket := range summary.BySport {
		if bucket.Key == "bike" && bucket.ActivityCount != 1 {
			t.Fatalf("bike facet = %+v", bucket)
		}
		if bucket.Key == "run" && bucket.ActivityCount != 2 {
			t.Fatalf("run facet = %+v", bucket)
		}
	}
}

func openActivitiesIntegrationDB(t *testing.T) *gorm.DB {
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
