//go:build integration

package sleep

import (
	"os"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPeriodReportUsesWakeDateAndMainSleepMetrics(t *testing.T) {
	db := openSleepIntegrationDB(t)
	rawID := uuid.New()
	createdAt := time.Now().UTC()
	if err := db.Create(&models.RawFile{ID: rawID, SHA256: "sleep-period-report-" + rawID.String(), OriginalFilename: "sleep.xml", StoragePath: "/tmp/sleep.xml", SourceKind: "apple_health_export", UploadedVia: "test", CreatedAt: createdAt}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	t.Cleanup(func() { db.Exec("delete from tb_raw_files where id = ?", rawID) })

	main := models.SleepSession{ID: uuid.New(), WakeDate: time.Date(2026, time.February, 2, 0, 0, 0, 0, time.UTC), TimeInBedS: 28800, AsleepS: 25200, Efficiency: 0.875, IsMainSleep: true, CoreS: 12000, DeepS: 6000, RemS: 7200, AwakeS: 3600, Source: "Watch", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt}
	nap := models.SleepSession{ID: uuid.New(), WakeDate: time.Date(2026, time.February, 2, 0, 0, 0, 0, time.UTC), TimeInBedS: 3600, AsleepS: 1800, Efficiency: 0.5, IsMainSleep: false, Source: "Watch", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt}
	outside := models.SleepSession{ID: uuid.New(), WakeDate: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC), IsMainSleep: true, Source: "Watch", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt}
	for _, row := range []models.SleepSession{main, nap, outside} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create sleep session: %v", err)
		}
		t.Cleanup(func() { db.Exec("delete from tb_sleep_sessions where id = ?", row.ID) })
	}

	result, err := NewService(db).PeriodReport(PeriodFilters{From: time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("period report: %v", err)
	}
	if result.SessionCount != 2 || result.MainSleepCount != 1 || result.NapCount != 1 || result.AverageAsleepS != 25200 || result.StageSeconds.Deep != 6000 {
		t.Fatalf("sleep period report = %+v", result)
	}
	lifetime, err := NewService(db).Aggregates(AggregateFilters{
		From:        ptrDate(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)),
		To:          ptrDate(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)),
		Granularity: "lifetime",
	})
	if err != nil {
		t.Fatalf("lifetime aggregates: %v", err)
	}
	if len(lifetime) != 1 || lifetime[0].SessionCount != 2 || lifetime[0].MainSleepCount != 1 || lifetime[0].NapCount != 1 {
		t.Fatalf("lifetime aggregate = %+v", lifetime)
	}
}

func ptrDate(value time.Time) *time.Time { return &value }

func TestPeriodQueriesPreserveNonUTCCalendarBoundaries(t *testing.T) {
	db := openSleepIntegrationDB(t)
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	rawID := uuid.New()
	createdAt := time.Now().UTC()
	if err := db.Create(&models.RawFile{ID: rawID, SHA256: "sleep-period-boundary-" + rawID.String(), OriginalFilename: "sleep.xml", StoragePath: "/tmp/sleep.xml", SourceKind: "apple_health_export", UploadedVia: "test", CreatedAt: createdAt}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	t.Cleanup(func() { db.Exec("delete from tb_raw_files where id = ?", rawID) })

	rows := []models.SleepSession{
		{ID: uuid.New(), WakeDate: time.Date(2099, time.January, 31, 0, 0, 0, 0, time.UTC), AsleepS: 111, IsMainSleep: true, FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), WakeDate: time.Date(2099, time.February, 1, 0, 0, 0, 0, time.UTC), AsleepS: 222, IsMainSleep: true, FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), WakeDate: time.Date(2099, time.February, 28, 0, 0, 0, 0, time.UTC), AsleepS: 333, IsMainSleep: true, FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), WakeDate: time.Date(2099, time.March, 1, 0, 0, 0, 0, time.UTC), AsleepS: 444, IsMainSleep: true, FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt},
	}
	for _, row := range rows {
		row := row
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create sleep session: %v", err)
		}
		t.Cleanup(func() { db.Exec("delete from tb_sleep_sessions where id = ?", row.ID) })
	}

	filters := PeriodFilters{
		From: time.Date(2099, time.February, 1, 0, 0, 0, 0, tokyo),
		To:   time.Date(2099, time.March, 1, 0, 0, 0, 0, tokyo),
	}
	service := NewService(db)
	values, err := service.PeriodSessions(filters)
	if err != nil {
		t.Fatalf("period sessions: %v", err)
	}
	if len(values) != 2 || !values[0].WakeDate.Equal(rows[1].WakeDate) || !values[1].WakeDate.Equal(rows[2].WakeDate) {
		t.Fatalf("period sessions = %+v, want wake dates %v and %v", values, rows[1].WakeDate, rows[2].WakeDate)
	}

	report, err := service.PeriodReport(filters)
	if err != nil {
		t.Fatalf("period report: %v", err)
	}
	if report.SessionCount != 2 || report.MainSleepCount != 2 || report.AverageAsleepS != 277.5 {
		t.Fatalf("period report = %+v, want two sessions averaging 277.5 seconds", report)
	}
}

func openSleepIntegrationDB(t *testing.T) *gorm.DB {
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
