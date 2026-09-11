//go:build integration

package main

import (
	"os"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestIntegrationConfiguredMediaScheduleIsIdempotentAndOptOutable(t *testing.T) {
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

	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncAniList, true, "12h"); err != nil {
		t.Fatalf("enable AniList schedule: %v", err)
	}
	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncAniList, true, "12h"); err != nil {
		t.Fatalf("reconcile AniList schedule: %v", err)
	}
	var schedule models.JobSchedule
	if err := db.Where("kind = ?", jobs.KindMediaSyncAniList).First(&schedule).Error; err != nil {
		t.Fatalf("load AniList schedule: %v", err)
	}
	if !schedule.Enabled || schedule.ScheduleExpr != "12h0m0s" {
		t.Fatalf("schedule = %#v, want enabled 12h0m0s", schedule)
	}
	var count int64
	if err := db.Model(&models.JobSchedule{}).Where("kind = ?", jobs.KindMediaSyncAniList).Count(&count).Error; err != nil {
		t.Fatalf("count AniList schedules: %v", err)
	}
	if count != 1 {
		t.Fatalf("AniList schedules = %d, want 1", count)
	}

	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncAniList, true, "off"); err != nil {
		t.Fatalf("opt out AniList schedule: %v", err)
	}
	if err := db.First(&schedule, "kind = ?", jobs.KindMediaSyncAniList).Error; err != nil {
		t.Fatalf("reload opted-out schedule: %v", err)
	}
	if schedule.Enabled {
		t.Fatal("opted-out schedule remained enabled")
	}
	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncBangumi, false, ""); err != nil {
		t.Fatalf("skip unconfigured Bangumi schedule: %v", err)
	}
	if err := db.Model(&models.JobSchedule{}).Where("kind = ?", jobs.KindMediaSyncBangumi).Count(&count).Error; err != nil {
		t.Fatalf("count Bangumi schedules: %v", err)
	}
	if count != 0 {
		t.Fatalf("unconfigured Bangumi schedules = %d, want 0", count)
	}
}
